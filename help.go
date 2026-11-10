package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// shortHints is the last thing dropped: whatever else does not fit, the user
// still needs to be told where the full list is.
const shortHints = "? keys "

// transportHints is the one-line reminder in the corner of the transport bar.
func transportHints(focus int) string {
	switch focus {
	case focusRecent:
		return "←→ covers · ↑↓ rows · ↵ play · / search · ? keys "
	case focusPlayer:
		return "←→ seek · ↑↓ volume · n/p track · v viz · t theme · ? keys "
	}
	return "↑↓ select · ↵ play · / search · space pause · ? keys "
}

// fitHints picks the longest form that fits whole in the space left beside the
// track title. Truncating instead would cut mid-word and, worse, silently.
func fitHints(hints string, room int) string {
	if lipgloss.Width(hints) <= room {
		return hints
	}
	if lipgloss.Width(shortHints) <= room {
		return shortHints
	}
	return ""
}

type helpEntry struct{ keys, action string }

type helpSection struct {
	title   string
	entries []helpEntry
}

var helpSections = []helpSection{
	{"SCREENS", []helpEntry{
		{"1 / 2", "Library / Focus"}, {"3 / 4", "Visualizer / Lyrics"}, {"c / esc", "queue / close"}, {"[ / ]", "library category"},
	}},
	{"PLAYBACK", []helpEntry{
		{"space", "play / pause"},
		{"n / p", "next / prev track"},
		{"← →", "seek −5 s / +5 s"},
		{"↑ ↓", "volume up / down"},
		{"s", "shuffle on / off"},
		{"r", "cycle repeat"},
	}},
	{"BROWSING", []helpEntry{
		{"/", "search"},
		{"tab/S-tab", "focus next / prev"},
		{"j k / ↑ ↓", "move selection"},
		{"↵", "play selection"},
		{"← →", "prev / next cover"},
	}},
	{"IN SEARCH", []helpEntry{
		{"↵", "search / play"},
		{"tab / esc", "next cat / close"},
		{"a / A", "queue end / next"},
	}},
	{"LOOK", []helpEntry{
		{"v", "bars/torus/sphere"},
		{"t", "cycle theme"},
		{"Q", "equalizer"},
		{"P / 0", "EQ preset / reset"},
		{"h l / ← →", "EQ select band"},
		{"j k / ↓ ↑", "EQ gain − / +"},
	}},
	{"OTHER", []helpEntry{
		{"?", "this help"},
		{"a", "account, sign out"},
		{"R", "reload web player"},
		{"q / ctrl+c", "quit"},
	}},
}

// Help is also the mouse command menu. Pages retain every action at 40×12.
// Navigation keys page it; other keys keep the original dismiss-only behavior.
type helpLine struct{ text, key, section string }

func helpLines() []helpLine {
	var out []helpLine
	for _, s := range helpSections {
		out = append(out, helpLine{text: s.title})
		for _, e := range s.entries {
			out = append(out, helpLine{text: pad(e.keys, 10) + " " + e.action, key: e.keys, section: s.title})
		}
	}
	return out
}
func (m model) helpGeometry() (cols, rows, colW, pages int) {
	cols = 1
	if m.w >= 70 {
		cols = 2
	}
	rows = max(1, m.h-4)
	colW = max(1, (m.w-2)/cols)
	pages = (len(helpLines()) + cols*rows - 1) / (cols * rows)
	return
}
func (m model) helpTargets() []keyTarget {
	cols, rows, colW, pages := m.helpGeometry()
	page := min(m.helpPage, pages-1)
	entries := helpLines()
	var targets []keyTarget
	for c := 0; c < cols; c++ {
		for r := 0; r < rows; r++ {
			i := page*rows*cols + c*rows + r
			if i < len(entries) && entries[i].key != "" {
				e := entries[i]
				if e.section == "IN SEARCH" || e.section == "BROWSING" {
					continue
				}
				tokens := helpKeyTokens(e.key)
				x := 1 + c*colW
				first := ""
				for _, token := range tokens {
					if token != "/" {
						key := token
						if e.section == "PLAYBACK" {
							key = "player:" + key
						}
						if strings.Contains(e.text, " EQ ") {
							key = "eq:" + key
						}
						if first == "" {
							first = key
						}
						targets = append(targets, keyTarget{rect{x, 2 + r, lipgloss.Width(token), 1}, key, token})
					}
					x += lipgloss.Width(token) + 1
				}
				targets = append(targets, keyTarget{rect{1 + c*colW + 11, 2 + r, max(0, colW-11), 1}, first, e.text})
			}
		}
	}
	return clipTargets(targets, m.w, m.h)
}
func (m model) updateHelp(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	_, _, _, pages := m.helpGeometry()
	switch msg.String() {
	case "right", "down", "pgdown", "tab":
		m.helpPage = (m.helpPage + 1) % pages
	case "left", "up", "pgup", "shift+tab":
		m.helpPage = (m.helpPage + pages - 1) % pages
	default:
		m.helpOpen = false
	}
	return m, nil
}
func (m model) helpView() string {
	cols, rows, colW, pages := m.helpGeometry()
	entries := helpLines()
	page := min(m.helpPage, pages-1)
	lines := []string{pad("PISKIN MUSIC · KEYS", m.w), pad("click action · arrows page · other key closes", m.w)}
	for r := 0; r < rows; r++ {
		line := " "
		for c := 0; c < cols; c++ {
			i := page*rows*cols + c*rows + r
			text := ""
			if i < len(entries) {
				text = entries[i].text
				if entries[i].key == "" {
					text = lipgloss.NewStyle().Foreground(accentHi).Bold(true).Render(text)
				} else {
					text = lipgloss.NewStyle().Foreground(fgDim).Render(text)
				}
			}
			line += pad(text, colW)
		}
		lines = append(lines, pad(line, m.w))
	}
	lines = append(lines, pad(fmt.Sprintf("‹ prev    %d/%d    next ›   · click here to close", page+1, pages), m.w))
	return strings.Join(lines, "\n")
}
