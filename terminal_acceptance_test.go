package main

import (
	"os"
	"sync/atomic"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type terminalPlayback struct {
	playback
	pauses atomic.Int32
	seeks  atomic.Int32
	closed atomic.Bool
}

func (p *terminalPlayback) PlayPause() error           { p.pauses.Add(1); return nil }
func (p *terminalPlayback) SeekTo(time.Duration) error { p.seeks.Add(1); return nil }
func (p *terminalPlayback) Close()                     { p.closed.Store(true) }

// Use the production Update/View and input parser without opening a browser or
// capture device. The PTY driver exercises actual key/mouse/focus/resize bytes.
type terminalAcceptance struct {
	model
	visited                                  [4]bool
	helpSeen, blurSeen, focusSeen, smallSeen bool
}

func (m terminalAcceptance) Init() tea.Cmd { return m.commands().tick() }
func (m terminalAcceptance) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := m.model.Update(msg)
	m.model = next.(model)
	m.visited[m.screen] = true
	m.helpSeen = m.helpSeen || m.helpOpen
	m.smallSeen = m.smallSeen || (m.w > 0 && m.w < 40)
	switch msg.(type) {
	case tea.BlurMsg:
		m.blurSeen = true
	case tea.FocusMsg:
		m.focusSeen = true
	}
	return m, cmd
}

func TestTerminalAcceptance(t *testing.T) {
	if os.Getenv("AMTUI_TERMINAL_ACCEPTANCE") != "1" {
		t.Skip("run through scripts/terminal-acceptance.py with a test binary")
	}
	m := performanceModel(screenLibrary, true, false)
	p := &terminalPlayback{}
	m.eng, m.cmds.tick = p, nil
	program := tea.NewProgram(terminalAcceptance{model: m}, tea.WithAltScreen(), tea.WithMouseCellMotion(), tea.WithReportFocus())
	final, err := program.Run()
	if err != nil {
		t.Fatal(err)
	}
	got := final.(terminalAcceptance)
	if got.visited != [4]bool{true, true, true, true} || !got.helpSeen || !got.blurSeen || !got.focusSeen || !got.smallSeen {
		t.Fatalf("terminal paths missed: screens=%v help=%t blur=%t focus=%t small=%t", got.visited, got.helpSeen, got.blurSeen, got.focusSeen, got.smallSeen)
	}
	if p.pauses.Load() != 2 || p.seeks.Load() != 1 || !p.closed.Load() {
		t.Fatalf("terminal commands: pause=%d seek=%d closed=%t", p.pauses.Load(), p.seeks.Load(), p.closed.Load())
	}
	t.Log("PTY keyboard/mouse/focus/resize acceptance passed using fixture playback")
}
