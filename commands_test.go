package main

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/k1y0miiii/applemusic-tui/engine"
	"github.com/k1y0miiii/applemusic-tui/lyrics"
	"github.com/k1y0miiii/applemusic-tui/mpris"
	"github.com/k1y0miiii/applemusic-tui/visualizer"
)

// Unexpected operations panic through the embedded interface. These tests run
// real root commands against only the small portion of playback they exercise.
type commandPlayback struct {
	playback
	calls []string
	err   error
	st    engine.State
}

func (p *commandPlayback) call(s string) error          { p.calls = append(p.calls, s); return p.err }
func (p *commandPlayback) State() (engine.State, error) { return p.st, p.call("state") }
func (p *commandPlayback) Library() (engine.SearchResults, error) {
	return engine.SearchResults{}, p.call("library")
}
func (p *commandPlayback) Search(s string) (engine.SearchResults, error) {
	return engine.SearchResults{}, p.call("search:" + s)
}
func (p *commandPlayback) PlayPause() error             { return p.call("play-pause") }
func (p *commandPlayback) Play(kind, id string) error   { return p.call("play:" + kind + ":" + id) }
func (p *commandPlayback) JumpTo(i int) error           { return p.call(fmt.Sprintf("jump:%d", i)) }
func (p *commandPlayback) SeekTo(d time.Duration) error { return p.call("seek:" + d.String()) }
func (p *commandPlayback) Close()                       { _ = p.call("close") }

func TestPlaybackCommandsAreDeferredAndKeepTheirRequest(t *testing.T) {
	p, other := &commandPlayback{st: engine.State{Volume: 42}}, &commandPlayback{}
	m := model{eng: p, sQuery: "first"}
	state, search := m.fetchState(), m.searchCmd()
	m.eng, m.sQuery = other, "second"
	if len(p.calls) != 0 {
		t.Fatal("building commands performed playback work")
	}
	if msg := state().(stateMsg); msg.st.Volume != 42 || msg.err != nil {
		t.Fatalf("state = %+v", msg)
	}
	if msg := search().(searchMsg); msg.err != nil || msg.keepInput {
		t.Fatalf("search = %+v", msg)
	}
	if !reflect.DeepEqual(p.calls, []string{"state", "search:first"}) || len(other.calls) != 0 {
		t.Fatalf("calls = %v; replacement calls = %v", p.calls, other.calls)
	}
}

func TestPlaybackReadCommandsPreserveFailures(t *testing.T) {
	want := errors.New("engine unavailable")
	p := &commandPlayback{err: want}
	m := model{eng: p, sQuery: "query"}
	if msg := m.fetchState()().(stateMsg); !errors.Is(msg.err, want) {
		t.Fatalf("state error = %v", msg.err)
	}
	if msg := m.searchCmd()().(searchMsg); !errors.Is(msg.err, want) || msg.keepInput {
		t.Fatalf("search failure = %+v", msg)
	}
	if msg := m.libraryCmd()().(searchMsg); !errors.Is(msg.err, want) || !msg.keepInput {
		t.Fatalf("library failure = %+v", msg)
	}
}

func TestPlaybackCommandFailuresReduceIntoAppState(t *testing.T) {
	for _, input := range []string{"keyboard", "queue click", "recent click"} {
		t.Run(input, func(t *testing.T) {
			p := &commandPlayback{err: errors.New("action rejected")}
			m := recentModel()
			m.eng = p
			var next tea.Model
			var cmd tea.Cmd
			switch input {
			case "keyboard":
				next, cmd = m.Update(tea.KeyMsg{Type: tea.KeySpace})
			case "queue click":
				next, cmd = m.Update(tea.MouseMsg{X: 5, Y: m.layout().queueTop(), Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
			case "recent click":
				l := m.layout()
				next, cmd = m.Update(tea.MouseMsg{X: 1 + l.recentGrid.gap, Y: l.recentTop(), Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
			}
			if cmd == nil || len(p.calls) != 0 {
				t.Fatal("input must defer playback to a command")
			}
			msg := cmd()
			if msg != noteMsg("action rejected") || len(p.calls) != 1 {
				t.Fatalf("message = %v; calls = %v", msg, p.calls)
			}
			failed, _ := next.Update(msg)
			got := failed.(model)
			if got.note != "action rejected" || got.loading != "" || got.audioInitializing() {
				t.Fatalf("failure left pending state: %+v", got)
			}
		})
	}
}

func TestInjectedStartupAndReconnectCloseBeforeConnecting(t *testing.T) {
	old, replacement := &commandPlayback{}, &commandPlayback{}
	m := model{phase: phaseReady, eng: old, statusCh: make(chan string, 1)}
	m.cmds.connect = func(ch chan string) tea.Cmd {
		if ch != m.statusCh {
			t.Fatal("connection lost the status channel")
		}
		return func() tea.Msg {
			if !reflect.DeepEqual(old.calls, []string{"close"}) {
				t.Fatalf("connect before close: %v", old.calls)
			}
			return readyMsg{replacement}
		}
	}
	next, cmd := m.reconnect("reconnecting")
	boot := next.(model)
	if boot.eng != nil || boot.phase != phaseBoot || len(old.calls) != 0 {
		t.Fatal("reconnect did not defer closing")
	}
	if msg := cmd().(readyMsg); msg.eng != replacement {
		t.Fatal("injected connection ignored")
	}
	// A connection failure follows the same reducer path as production.
	boot.cmds.connect = func(chan string) tea.Cmd { return func() tea.Msg { return failMsg{errors.New("offline")} } }
	boot.cmds.tick = func() tea.Cmd { return func() tea.Msg { return tickMsg{} } }
	batch := boot.Init()().(tea.BatchMsg)
	failed, _ := boot.Update(batch[1]())
	if got := failed.(model); got.phase != phaseFail || got.status != "offline" {
		t.Fatalf("startup failure = %+v", got)
	}
}

func TestInjectedReadyCommandsAndPollingKeepExistingCadence(t *testing.T) {
	p := &commandPlayback{st: engine.State{Authed: true}}
	opens, polls, ticks := 0, 0, 0
	m := model{cmds: commandSet{
		openVisualizer: func() tea.Cmd {
			return func() tea.Msg { opens++; return visualizerOpenedMsg{err: errors.New("no capture")} }
		},
		publishMPRIS: func(mpris.Controls) (*mpris.Server, error) { return nil, errors.New("no bus") },
		poll:         func() tea.Cmd { return func() tea.Msg { polls++; return pollMsg{} } },
		tick:         func() tea.Cmd { return func() tea.Msg { ticks++; return tickMsg{} } },
	}}
	next, cmd := m.Update(readyMsg{p})
	m = next.(model)
	batch := cmd().(tea.BatchMsg)
	if len(batch) != 3 || len(p.calls) != 0 || opens != 0 {
		t.Fatal("ready changed startup work")
	}
	for _, c := range batch {
		next, _ = m.Update(c())
		m = next.(model)
	}
	if m.captureStatus() != captureUnavailable || opens != 1 {
		t.Fatal("capture failure not represented")
	}
	// One state result schedules one poll; a poll executes one state read.
	_, cmd = m.Update(stateMsg{st: p.st})
	pollMessage := cmd() // only a poll remains, so Bubble Tea needn't return a batch
	next, cmd = m.Update(pollMessage)
	m = next.(model)
	_ = cmd()
	if polls != 1 || !reflect.DeepEqual(p.calls, []string{"state", "library", "state"}) {
		t.Fatalf("polls=%d calls=%v", polls, p.calls)
	}
	_, cmd = m.Update(tickMsg{})
	_ = cmd()
	if ticks != 1 || len(p.calls) != 3 || opens != 1 {
		t.Fatal("UI tick performed additional playback/capture work")
	}
	// Reconnecting with a service keeps it; ready must not reopen capture.
	m.vizService = &visualizer.Service{}
	next, cmd = m.Update(readyMsg{p})
	if len(cmd().(tea.BatchMsg)) != 2 || next.(model).vizService != m.vizService {
		t.Fatal("ready replaced capture")
	}
}

func TestInjectedMetadataCommandsRetainTrackIdentity(t *testing.T) {
	var requested []string
	m := model{phase: phaseReady, lyFor: "old", artFor: "old", cmds: commandSet{
		poll: func() tea.Cmd { return nil },
		lyrics: func(id, artist, title string, d time.Duration) tea.Cmd {
			requested = append(requested, id+":"+artist+":"+title+":"+d.String())
			return func() tea.Msg { return lyricsMsg{id: id, ly: lyrics.Lyrics{}} }
		},
		artwork: func(id, template string) tea.Cmd { return func() tea.Msg { return artMsg{id: id} } },
	}}
	st := engine.State{Authed: true, Now: engine.Track{ID: "new", Artist: "artist", Title: "title", Duration: time.Minute, Art: "cover"}}
	next, cmd := m.Update(stateMsg{st: st})
	m = next.(model)
	if !reflect.DeepEqual(requested, []string{"new:artist:title:1m0s"}) {
		t.Fatalf("requests = %v", requested)
	}
	// Move on before these commands complete. Late metadata must stay ignored.
	m.lyFor, m.artFor = "newer", "newer"
	for _, c := range cmd().(tea.BatchMsg) {
		next, _ = m.Update(c())
		m = next.(model)
	}
	if m.lyFor != "newer" || m.artFor != "newer" {
		t.Fatal("late response changed track identity")
	}
}
