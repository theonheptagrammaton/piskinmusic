package main

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/k1y0miiii/applemusic-tui/engine"
	"github.com/k1y0miiii/applemusic-tui/lyrics"
	"github.com/k1y0miiii/applemusic-tui/mpris"
	"github.com/k1y0miiii/applemusic-tui/visualizer"
)

// commandSet supplies per-model command factories and desktop publication.
// Factories build commands whose effects run when Bubble Tea executes them;
// MPRIS publication retains its existing synchronous startup behavior. Nil fields
// use production defaults. Playback effects are injected through playback.
type commandSet struct {
	connect        func(chan string) tea.Cmd
	openVisualizer func() tea.Cmd
	lyrics         func(string, string, string, time.Duration) tea.Cmd
	artwork        func(string, string) tea.Cmd
	tileArt        func(string, string) tea.Cmd
	tick           func() tea.Cmd
	poll           func() tea.Cmd
	publishMPRIS   func(mpris.Controls) (*mpris.Server, error)
}

func (m model) commands() commandSet {
	c := m.cmds
	if c.connect == nil {
		c.connect = connectCmd
	}
	if c.openVisualizer == nil {
		c.openVisualizer = func() tea.Cmd { return openVisualizerCmd(visualizer.OpenSystemSource) }
	}
	if c.lyrics == nil {
		c.lyrics = fetchLyricsCmd
	}
	if c.artwork == nil {
		c.artwork = fetchArtworkCmd
	}
	if c.tileArt == nil {
		c.tileArt = fetchTileArtCmd
	}
	if c.tick == nil {
		interval := m.tickInterval()
		c.tick = func() tea.Cmd {
			return tea.Tick(interval, func(t time.Time) tea.Msg { return tickMsg(t) })
		}
	}
	if c.poll == nil {
		c.poll = poll
	}
	if c.publishMPRIS == nil {
		c.publishMPRIS = mpris.Publish
	}
	return c
}

func poll() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg { return pollMsg{} })
}

func closeServiceAsync(service *visualizer.Service) {
	if service != nil {
		go func() {
			_ = service.Close()
		}()
	}
}

// The old browser must be gone before the new one starts: both use the same
// profile directory, and Chrome holds a lock on it.
func reconnectCmd(eng playback, connect tea.Cmd) tea.Cmd {
	return func() tea.Msg {
		if eng != nil {
			eng.Close()
		}
		return connect()
	}
}

func connectCmd(ch chan string) tea.Cmd {
	return func() tea.Msg {
		eng, err := engine.Connect(func(s string) { ch <- s })
		if err != nil {
			return failMsg{err}
		}
		return readyMsg{eng}
	}
}

func listenStatus(ch chan string) tea.Cmd {
	return func() tea.Msg { return statusMsg(<-ch) }
}

func (m model) fetchState() tea.Cmd {
	eng := m.eng
	return func() tea.Msg {
		st, err := eng.State()
		return stateMsg{st, err}
	}
}

func openVisualizerCmd(openSource func() (visualizer.Source, error)) tea.Cmd {
	return func() tea.Msg {
		source, err := openSource()
		if err != nil {
			return visualizerOpenedMsg{err: err}
		}
		sourceName := ""
		if source != nil {
			sourceName = source.Name()
		}
		service, err := visualizer.NewService(source)
		if err != nil {
			if source != nil {
				_ = source.Close()
			}
			return visualizerOpenedMsg{err: err}
		}
		return visualizerOpenedMsg{service: service, source: sourceName}
	}
}

func closeVisualizerCmd(service *visualizer.Service) tea.Cmd {
	return func() tea.Msg {
		_ = service.Close()
		return nil
	}
}

func doCmd(f func() error) tea.Cmd {
	return func() tea.Msg {
		if err := f(); err != nil {
			return noteMsg(err.Error())
		}
		return nil
	}
}

func fetchLyricsCmd(id, artist, title string, dur time.Duration) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		ly, err := lyrics.Fetch(ctx, artist, title, dur)
		if err != nil {
			ly = lyrics.Lyrics{} // network miss = just "no lyrics", not an error banner
		}
		return lyricsMsg{id: id, ly: ly}
	}
}

func fetchArtworkCmd(id, template string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), artworkTimeout)
		defer cancel()
		img, err := fetchArtwork(ctx, artworkURL(template, artworkPx))
		if err != nil {
			return artMsg{id: id} // nil image: the panel just shows lyrics only
		}
		return artMsg{id: id, img: img}
	}
}

// tileArtPx is smaller than the lyrics cover: grid tiles are a dozen cells wide,
// and ten of them download at once.
const tileArtPx = 128

func fetchTileArtCmd(id, template string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), artworkTimeout)
		defer cancel()
		img, err := fetchArtwork(ctx, artworkURL(template, tileArtPx))
		if err != nil {
			return tileArtMsg{id: id} // nil image: the tile falls back to a placeholder
		}
		return tileArtMsg{id: id, img: img}
	}
}

// fetchTileArtCmds fires one fetch per entry that has an artwork template and no
// cover yet. Bubble Tea runs the batch concurrently.
func (m model) fetchTileArtCmds() []tea.Cmd {
	var cmds []tea.Cmd
	for _, tr := range m.recent {
		if tr.Art == "" {
			continue
		}
		if _, done := m.tileArt[tr.ID]; done {
			continue
		}
		cmds = append(cmds, m.commands().tileArt(tr.ID, tr.Art))
	}
	return cmds
}

func (m model) searchCmd() tea.Cmd {
	eng, term := m.eng, m.sQuery
	return func() tea.Msg {
		res, err := eng.Search(term)
		return searchMsg{res: res, err: err}
	}
}

func (m model) libraryCmd() tea.Cmd {
	eng := m.eng
	return func() tea.Msg {
		res, err := eng.Library()
		return searchMsg{res: res, err: err, keepInput: true}
	}
}

func accountCmd(eng playback) tea.Cmd {
	return func() tea.Msg {
		acct, err := eng.Account()
		return accountMsg{acct: acct, err: err}
	}
}

func signOutCmd(eng playback) tea.Cmd {
	return func() tea.Msg {
		return signedOutMsg{err: eng.SignOut()}
	}
}
