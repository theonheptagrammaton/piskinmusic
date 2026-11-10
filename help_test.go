package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Retain the original 70×22 dashboard regression alongside new 40×12 coverage.
const (
	minCols = 70
	minRows = 22
)

func TestHelpOverlayFitsTheSmallestTerminal(t *testing.T) {
	defer lipgloss.SetColorProfile(lipgloss.ColorProfile())
	lipgloss.SetColorProfile(termenv.Ascii)

	m := model{w: minCols, h: minRows}
	lines := strings.Split(m.helpView(), "\n")
	if len(lines) > minRows {
		t.Errorf("help is %d rows in a %d-row terminal — it would scroll away",
			len(lines), minRows)
	}
	for i, line := range lines {
		if got := lipgloss.Width(line); got > minCols {
			t.Errorf("help row %d is %d cells wide, over the %d-column minimum",
				i, got, minCols)
		}
	}
}

func TestHelpShowsEveryEntryWhole(t *testing.T) {
	// The whole point of the overlay is that nothing gets cut off. pad truncates
	// silently, so a longer entry would quietly lose its tail rather than fail.
	defer lipgloss.SetColorProfile(lipgloss.ColorProfile())
	lipgloss.SetColorProfile(termenv.Ascii)

	out := model{w: minCols, h: minRows}.helpView()
	for _, section := range helpSections {
		if !strings.Contains(out, section.title) {
			t.Errorf("section %q missing from the overlay", section.title)
		}
		for _, e := range section.entries {
			if !strings.Contains(out, e.action) {
				t.Errorf("entry %q (%s) is truncated at %d columns", e.action, e.keys, minCols)
			}
		}
	}
}

func TestTransportHintsAreNeverCutInHalf(t *testing.T) {
	// This is the defect the overlay exists for, and measuring the row width
	// will not find it: transportPanel pads every row to the panel, so an
	// overlong hint line comes back the right width with its tail gone. Check
	// the text instead — whichever form is chosen has to appear whole.
	defer lipgloss.SetColorProfile(lipgloss.ColorProfile())
	lipgloss.SetColorProfile(termenv.Ascii)

	for _, w := range []int{minCols - 2, 100, 160} {
		for _, focus := range []int{focusQueue, focusRecent, focusPlayer} {
			out := model{w: w + 2, h: minRows, focus: focus}.transportPanel(w)
			full := strings.TrimSpace(transportHints(focus))
			switch {
			case strings.Contains(out, full):
			case strings.Contains(out, strings.TrimSpace(shortHints)):
			default:
				t.Errorf("w=%d focus=%d: neither %q nor %q survived the render:\n%s",
					w, focus, full, strings.TrimSpace(shortHints), out)
			}
		}
	}
}

func TestFitHintsFallsBackRatherThanTruncating(t *testing.T) {
	full := transportHints(focusPlayer)
	if got := fitHints(full, lipgloss.Width(full)); got != full {
		t.Errorf("exact fit was rejected: %q", got)
	}
	if got := fitHints(full, lipgloss.Width(full)-1); got != shortHints {
		t.Errorf("one cell short gave %q, want the short form", got)
	}
	if got := fitHints(full, lipgloss.Width(shortHints)-1); got != "" {
		t.Errorf("no room gave %q, want nothing at all", got)
	}
}

func TestHelpKeyOpensAndAnyKeyCloses(t *testing.T) {
	m := model{phase: phaseReady, w: minCols, h: minRows}
	next, _ := m.updateKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	opened := next.(model)
	if !opened.helpOpen {
		t.Fatal("? did not open the help overlay")
	}

	// Any key closes it, and must not also run its own command — reading the
	// list should never fire the thing you were looking up.
	opened.vizMode = vizBars
	closed, _ := opened.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	after := closed.(model)
	if after.helpOpen {
		t.Error("a key press did not dismiss the help overlay")
	}
	if after.vizMode != vizBars {
		t.Errorf("the dismissing key also cycled the visualizer to %d", after.vizMode)
	}
}

func TestHelpRetainsEveryActionAcrossNarrowPages(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {60, 20}, {80, 24}, {120, 32}} {
		m := model{w: size[0], h: size[1], phase: phaseReady, helpOpen: true}
		_, _, _, pages := m.helpGeometry()
		all := ""
		for range pages {
			out := m.helpView()
			all += out
			for _, line := range strings.Split(out, "\n") {
				if lipgloss.Width(line) > m.w {
					t.Fatalf("help overflows %v", size)
				}
			}
			next, _ := m.Update(keyMessage("right"))
			m = next.(model)
			if !m.helpOpen {
				t.Fatal("paging closed help")
			}
		}
		if m.helpPage != 0 {
			t.Fatal("paging did not wrap")
		}
		for _, section := range helpSections {
			for _, e := range section.entries {
				if !strings.Contains(all, e.action) {
					t.Fatalf("%v lost action %q", size, e.action)
				}
			}
		}
	}
}

func TestHelpMouseCanRunEachGeneralCommandAndBothScreenKeys(t *testing.T) {
	t.Setenv("AMTUI_CONFIG_DIR", t.TempDir())
	m := recentModel()
	m.w, m.h = 40, 12
	m.eng = &screenPlayback{}
	m.helpOpen = true
	_, _, _, pages := m.helpGeometry()
	seen := map[string]bool{}
	for page := 0; page < pages; page++ {
		m.helpPage = page
		for _, a := range m.helpTargets() {
			next, _ := m.Update(pressAt(a.x, a.y))
			got := next.(model)
			if a.key != "?" && got.helpOpen {
				t.Fatalf("help click %s did not dispatch", a.key)
			}
			if a.key == "4" && got.screen != screenLyrics {
				t.Fatal("second screen key in row is inert")
			}
			seen[a.key] = true
		}
	}
	for _, key := range []string{"1", "2", "3", "4", "c", "esc", "[", "]", "player:n", "player:p", "player:space", "player:←", "player:→", "player:↑", "player:↓", "player:s", "player:r", "v", "t", "Q", "eq:P", "eq:0", "a", "R", "q"} {
		if !seen[key] {
			t.Fatalf("mouse command %q missing", key)
		}
	}
}
