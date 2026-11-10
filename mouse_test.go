package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/k1y0miiii/applemusic-tui/engine"
)

func mouseModel(w, h, tracks int) model {
	q := make([]engine.Track, tracks)
	for i := range q {
		q[i] = engine.Track{
			Title:    fmt.Sprintf("Track%02d", i),
			Artist:   "Artist",
			Duration: 3 * time.Minute,
		}
	}
	return model{
		phase: phaseReady, w: w, h: h,
		st: engine.State{Queue: q, Dur: 4 * time.Minute, Pos: time.Minute},
	}
}

func click(m model, x, y int) model {
	next, _ := m.updateMouse(tea.MouseMsg{
		X: x, Y: y,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})
	return next.(model)
}

func TestClickSelectsTheTrackDrawnOnThatRow(t *testing.T) {
	// The point of the whole layout struct: the hit-test has to agree with what
	// is actually on screen. So find each title in the rendered frame and click
	// the row it landed on.
	defer lipgloss.SetColorProfile(lipgloss.ColorProfile())
	lipgloss.SetColorProfile(termenv.Ascii)

	m := mouseModel(100, 34, 40)
	rows := strings.Split(m.View(), "\n")

	checked := 0
	for want := 0; want < 40; want++ {
		title := fmt.Sprintf("Track%02d", want)
		for y, row := range rows {
			if !strings.Contains(row, title) {
				continue
			}
			if got := click(m, 5, y); got.selIdx != want {
				t.Errorf("%s is drawn on row %d, but clicking there selected %d",
					title, y, got.selIdx)
			}
			checked++
			break
		}
	}
	if checked < 5 {
		t.Fatalf("only %d titles were visible in the frame, too few to prove anything", checked)
	}
}

// barSpan finds the progress bar in a rendered frame by its glyphs, so the
// geometry is checked against the picture rather than against itself.
func barSpan(t *testing.T, m model) (first, last int) {
	t.Helper()
	rows := strings.Split(m.View(), "\n")
	line := []rune(rows[m.h-2])
	first, last = -1, -1
	for i, r := range line {
		if r == '─' || r == '━' || strings.ContainsRune("▁▂▃▄▅▆▇█", r) {
			// " ▄▄ " prefixes the line with two of the same block glyphs; the
			// bar is the long run, so skip anything in the first few cells.
			if i < 5 {
				continue
			}
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	if first < 0 {
		t.Fatalf("no progress bar found in %q", string(line))
	}
	return first, last
}

func TestProgressGeometryMatchesTheDrawnBar(t *testing.T) {
	defer lipgloss.SetColorProfile(lipgloss.ColorProfile())
	lipgloss.SetColorProfile(termenv.Ascii)

	for _, size := range [][2]int{{100, 34}, {70, 22}, {160, 48}} {
		m := mouseModel(size[0], size[1], 5)
		barX, barW := m.progressGeometry(m.w - 2)
		first, last := barSpan(t, m)
		// +1 for the panel border the bar sits inside.
		if first != barX+1 {
			t.Errorf("%dx%d: bar is drawn from column %d, geometry says %d",
				m.w, m.h, first, barX+1)
		}
		if got := last - first + 1; got != barW {
			t.Errorf("%dx%d: bar is %d cells wide, geometry says %d", m.w, m.h, got, barW)
		}
	}
}

func TestClickOnTheProgressBarSeeksThere(t *testing.T) {
	m := mouseModel(100, 34, 5)
	barX, barW := m.progressGeometry(m.w - 2)

	for _, frac := range []float64{0, 0.25, 0.5, 0.9} {
		x := 1 + barX + int(frac*float64(barW))
		got := click(m, x, m.h-2)
		want := time.Duration(frac * float64(m.st.Dur))
		// One cell of the bar is the resolution the user gets.
		if delta := got.st.Pos - want; delta > m.st.Dur/time.Duration(barW) || -delta > m.st.Dur/time.Duration(barW) {
			t.Errorf("click at %.0f%% of the bar seeked to %v, want about %v",
				frac*100, got.st.Pos, want)
		}
	}
}

func TestClickOffTheBarDoesNotSeek(t *testing.T) {
	m := mouseModel(100, 34, 5)
	barX, barW := m.progressGeometry(m.w - 2)
	for _, x := range []int{1, 1 + barX - 1, 1 + barX + barW, m.w - 2} {
		if got := click(m, x, m.h-2); got.st.Pos != m.st.Pos {
			t.Errorf("click at x=%d, outside the bar, moved the position to %v", x, got.st.Pos)
		}
	}
}

func TestWheelMovesTheQueueSelection(t *testing.T) {
	m := mouseModel(100, 34, 40)
	m.selIdx = 10
	lay := m.layout()

	wheel := func(m model, button tea.MouseButton) model {
		next, _ := m.updateMouse(tea.MouseMsg{
			X: 5, Y: lay.queueTop() + 1,
			Button: button, Action: tea.MouseActionPress,
		})
		return next.(model)
	}

	if got := wheel(m, tea.MouseButtonWheelDown); got.selIdx != 11 {
		t.Errorf("wheel down moved the selection to %d, want 11", got.selIdx)
	}
	if got := wheel(m, tea.MouseButtonWheelUp); got.selIdx != 9 {
		t.Errorf("wheel up moved the selection to %d, want 9", got.selIdx)
	}

	// And it must not run off either end.
	m.selIdx = 0
	if got := wheel(m, tea.MouseButtonWheelUp); got.selIdx != 0 {
		t.Errorf("wheel up at the top gave %d, want 0", got.selIdx)
	}
	m.selIdx = 39
	if got := wheel(m, tea.MouseButtonWheelDown); got.selIdx != 39 {
		t.Errorf("wheel down at the bottom gave %d, want 39", got.selIdx)
	}
}

func TestMouseIsIgnoredBehindOverlays(t *testing.T) {
	base := mouseModel(100, 34, 40)
	base.selIdx = 7
	for name, m := range map[string]model{
		"help":    func() model { c := base; c.helpOpen = true; return c }(),
		"search":  func() model { c := base; c.searchOpen = true; return c }(),
		"booting": func() model { c := base; c.phase = 0; return c }(),
	} {
		if got := click(m, 5, m.layout().queueTop()); got.selIdx != 7 {
			t.Errorf("%s: a click reached the queue underneath and selected %d", name, got.selIdx)
		}
	}
}

func TestLayoutMatchesTheRenderedFrame(t *testing.T) {
	// layout() exists so View and the hit-test cannot drift. Check its numbers
	// against a real frame rather than against themselves.
	defer lipgloss.SetColorProfile(lipgloss.ColorProfile())
	lipgloss.SetColorProfile(termenv.Ascii)

	for _, size := range [][2]int{{100, 34}, {70, 22}, {160, 48}} {
		m := mouseModel(size[0], size[1], 40)
		rows := strings.Split(m.View(), "\n")
		lay := m.layout()

		if len(rows) != m.h {
			t.Errorf("%dx%d: frame is %d rows", m.w, m.h, len(rows))
		}
		// The first queue entry has to be on the row layout says it is.
		if top := lay.queueTop(); top >= len(rows) || !strings.Contains(rows[top], "Track00") {
			t.Errorf("%dx%d: queueTop()=%d does not hold the first track: %q",
				m.w, m.h, top, rows[min(top, len(rows)-1)])
		}
	}
}

type screenPlayback struct{ commandPlayback }

func (p *screenPlayback) Next() error                      { return p.call("next") }
func (p *screenPlayback) Prev() error                      { return p.call("prev") }
func (p *screenPlayback) ToggleShuffle() error             { return p.call("shuffle") }
func (p *screenPlayback) CycleRepeat() error               { return p.call("repeat") }
func (p *screenPlayback) SetVolume(v int) error            { return p.call(fmt.Sprintf("volume:%d", v)) }
func (p *screenPlayback) SetEQ(g [10]int) error            { return p.call(fmt.Sprint("eq:", g)) }
func (p *screenPlayback) QueueNext(kind, id string) error  { return p.call("next:" + kind + ":" + id) }
func (p *screenPlayback) QueueLater(kind, id string) error { return p.call("later:" + kind + ":" + id) }
func (p *screenPlayback) Account() (engine.Account, error) {
	return engine.Account{}, p.call("account")
}
func (p *screenPlayback) SignOut() error { return p.call("sign-out") }

func pressAt(x, y int) tea.MouseMsg {
	return tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}
}

func TestLibraryMousePlaysTheVisibleScrolledRow(t *testing.T) {
	defer lipgloss.SetColorProfile(lipgloss.ColorProfile())
	lipgloss.SetColorProfile(termenv.Ascii)
	for _, size := range [][2]int{{40, 12}, {60, 20}, {80, 24}, {120, 32}, {180, 50}} {
		m := recentModel()
		m.w, m.h = size[0], size[1]
		p := &screenPlayback{}
		m.eng = p
		for i := 0; i < 35; i++ {
			m.library.Albums = append(m.library.Albums, engine.Track{ID: fmt.Sprint(i), Kind: "album", Title: fmt.Sprintf("Record%02d", i)})
		}
		m.librarySel[0] = 30
		l := m.layout()
		start := listWindow(30, 35, l.libraryBody.h)
		rows := strings.Split(m.View(), "\n")
		if !strings.Contains(rows[l.libraryBody.y], fmt.Sprintf("Record%02d", start)) {
			t.Fatalf("%v library scroll geometry differs from frame", size)
		}
		next, cmd := m.Update(pressAt(l.libraryBody.x, l.libraryBody.y))
		got := next.(model)
		if got.librarySel[0] != start || cmd == nil {
			t.Fatalf("%v click did not select drawn row", size)
		}
		cmd()
		if len(p.calls) != 1 || p.calls[0] != fmt.Sprintf("play:album:%d", start) {
			t.Fatalf("wrong record: %v", p.calls)
		}
	}
}

func TestSearchMouseScrollingPlayAndQueueActions(t *testing.T) {
	t.Setenv("AMTUI_CONFIG_DIR", t.TempDir())
	defer lipgloss.SetColorProfile(lipgloss.ColorProfile())
	lipgloss.SetColorProfile(termenv.Ascii)
	for _, size := range [][2]int{{40, 12}, {80, 24}, {180, 50}} {
		m := recentModel()
		m.w, m.h, m.searchOpen, m.sTab, m.sSel = size[0], size[1], true, 1, 25
		p := &screenPlayback{}
		m.eng = p
		for i := 0; i < 40; i++ {
			m.sRes.Songs = append(m.sRes.Songs, engine.Track{ID: fmt.Sprint(i), Kind: "song", Title: fmt.Sprintf("Result%02d", i)})
		}
		body, tabs, actions := m.searchGeometry()
		rows := strings.Split(m.View(), "\n")
		start := listWindow(25, 40, body.h)
		if !strings.Contains(rows[body.y], fmt.Sprintf("Result%02d", start)) {
			t.Fatal("selected search page is not visible")
		}
		next, cmd := m.Update(pressAt(body.x, body.y))
		if next.(model).sSel != start || cmd == nil {
			t.Fatal("search click missed scrolled result")
		}
		cmd()
		if p.calls[0] != fmt.Sprintf("play:song:%d", start) {
			t.Fatal(p.calls)
		}
		for _, action := range actions[1:3] {
			p.calls = nil
			_, cmd = m.Update(pressAt(action.x, action.y))
			if cmd == nil {
				t.Fatal("queue action inaccessible")
			}
			cmd()
			want := "later:song:25"
			if action.key == "A" {
				want = "next:song:25"
			}
			if len(p.calls) != 1 || p.calls[0] != want {
				t.Fatalf("%s: %v", action.label, p.calls)
			}
		}
		next, _ = m.Update(pressAt(tabs[2].x, tabs[2].y))
		if next.(model).sTab != 2 || next.(model).sSel != 0 {
			t.Fatal("search tab inaccessible")
		}
	}
}

func TestTransportEQAndAccountMouseUseKeyboardCommands(t *testing.T) {
	t.Setenv("AMTUI_CONFIG_DIR", t.TempDir())
	for _, size := range [][2]int{{40, 12}, {80, 24}, {120, 32}} {
		m := recentModel()
		m.w, m.h, m.screen, m.focus = size[0], size[1], screenFocus, focusPlayer
		p := &screenPlayback{}
		m.eng = p
		l := m.layout()
		next, cmd := m.Update(pressAt(l.play.x, l.play.y))
		if cmd == nil {
			t.Fatal("play icon is inert")
		}
		cmd()
		if next.(model).st.Playing == m.st.Playing || p.calls[0] != "play-pause" {
			t.Fatal("play icon differs from space")
		}
		_, cmd = m.Update(tea.MouseMsg{X: l.transport.x, Y: l.transport.y, Button: tea.MouseButtonWheelUp, Action: tea.MouseActionPress})
		cmd()
		if p.calls[1] != "volume:70" {
			t.Fatal("transport wheel did not change volume")
		}
		for _, a := range m.transportActions(l) {
			_, cmd = m.Update(pressAt(a.x, a.y))
			if cmd == nil {
				t.Fatal("transport target inert")
			}
			cmd()
		}
		m.eqOpen = true
		l = m.layout()
		rows := strings.Split(m.View(), "\n")
		for _, a := range m.eqActions(l) {
			if !strings.Contains(rows[a.y], a.label) {
				t.Fatalf("EQ action %q not drawn at hit row", a.label)
			}
			next, cmd = m.Update(pressAt(a.x, a.y))
			if a.key == "Q" {
				if next.(model).eqOpen {
					t.Fatal("EQ close inert")
				}
				continue
			}
			if cmd == nil {
				t.Fatal("EQ action inert")
			}
			cmd()
		}
		// A screen-label click cannot reach the content behind EQ.
		next, cmd = m.Update(pressAt(l.nav[3].x, l.nav[3].y))
		if next.(model).screen != m.screen || cmd != nil {
			t.Fatal("EQ allowed click-through")
		}
		m.eqOpen, m.acctOpen = false, true
		action := m.accountAction()
		if !strings.Contains(strings.Split(m.View(), "\n")[action.y], "Sign out") {
			t.Fatal("sign-out target not drawn")
		}
		_, cmd = m.Update(pressAt(action.x, action.y))
		if cmd == nil {
			t.Fatal("account sign-out inaccessible")
		}
		cmd()
		if p.calls[len(p.calls)-1] != "sign-out" {
			t.Fatal(p.calls)
		}
	}
}

func TestToolbarActionsReachOverlaysFromEveryScreen(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {80, 24}, {120, 32}} {
		for s := screenLibrary; s <= screenLyrics; s++ {
			m := recentModel()
			m.screen, m.w, m.h = s, size[0], size[1]
			for _, a := range m.layout().actions {
				next, _ := m.Update(pressAt(a.x, a.y))
				got := next.(model)
				switch a.key {
				case "/":
					if !got.searchOpen {
						t.Fatal("search inaccessible")
					}
				case "c":
					if !got.queueOpen {
						t.Fatal("queue inaccessible")
					}
				case "Q":
					if !got.eqOpen {
						t.Fatal("EQ inaccessible")
					}
				case "a":
					if !got.acctOpen {
						t.Fatal("account inaccessible")
					}
				case "?":
					if !got.helpOpen {
						t.Fatal("help inaccessible")
					}
				}
			}
		}
	}
}

func TestNoOverlayAcceptsOffscreenMouseInput(t *testing.T) {
	for _, size := range [][2]int{{0, 0}, {1, 1}, {20, 6}, {39, 11}, {40, 12}} {
		for _, overlay := range []string{"search", "help", "account", "EQ", "queue"} {
			m := recentModel()
			m.w, m.h = size[0], size[1]
			switch overlay {
			case "search":
				m.searchOpen = true
			case "help":
				m.helpOpen = true
			case "account":
				m.acctOpen = true
			case "EQ":
				m.eqOpen = true
			case "queue":
				m.toggleQueue()
			}
			for _, point := range [][2]int{{-1, 0}, {0, -1}, {m.w, 0}, {0, m.h}, {1000, 1000}} {
				next, cmd := m.Update(pressAt(point[0], point[1]))
				got := next.(model)
				if cmd != nil || got.helpOpen != m.helpOpen || got.acctOpen != m.acctOpen || got.searchOpen != m.searchOpen || got.eqOpen != m.eqOpen || got.queueOpen != m.queueOpen {
					t.Fatalf("%v %s accepted offscreen %v", size, overlay, point)
				}
			}
			_, tabs, actions := m.searchGeometry()
			targets := append(tabs, actions...)
			targets = append(targets, m.helpTargets()...)
			targets = append(targets, keyTarget{rect: m.accountAction()})
			for _, a := range targets {
				if a.w > 0 && a.h > 0 && (a.x < 0 || a.y < 0 || a.x+a.w > m.w || a.y+a.h > m.h) {
					t.Fatalf("%v offscreen overlay target %+v", size, a)
				}
			}
		}
	}
}
