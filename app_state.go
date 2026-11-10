package main

import (
	"fmt"
	"image"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/k1y0miiii/applemusic-tui/engine"
	"github.com/k1y0miiii/applemusic-tui/lastfm"
	"github.com/k1y0miiii/applemusic-tui/lyrics"
	"github.com/k1y0miiii/applemusic-tui/mpris"
	"github.com/k1y0miiii/applemusic-tui/visualizer"
)

// playback is the root application's view of the existing engine API. Engine
// snapshots stay value-owned; browser and MusicKit details remain in engine.
// Keep this limited to operations used by root commands and desktop controls.
type playback interface {
	State() (engine.State, error)
	PlayPause() error
	Next() error
	Prev() error
	ToggleShuffle() error
	CycleRepeat() error
	JumpTo(int) error
	SeekTo(time.Duration) error
	SetVolume(int) error
	SetEQ([10]int) error
	Play(kind, id string) error
	QueueNext(kind, id string) error
	QueueLater(kind, id string) error
	Search(string) (engine.SearchResults, error)
	Library() (engine.SearchResults, error)
	Account() (engine.Account, error)
	SignOut() error
	Reload() error
	Close()
}

var _ playback = (*engine.Engine)(nil)

type phase int

const (
	phaseBoot phase = iota
	phaseReady
	phaseFail
)

type (
	tickMsg   time.Time
	pollMsg   struct{}
	statusMsg string
	noteMsg   string
	readyMsg  struct{ eng playback }
	failMsg   struct{ err error }
	stateMsg  struct {
		st  engine.State
		err error
	}
	searchMsg struct {
		res       engine.SearchResults
		err       error
		keepInput bool // library load keeps the input focused
	}
	lyricsMsg struct {
		id string // track the lyrics belong to
		ly lyrics.Lyrics
	}
	artMsg struct {
		id  string // track the artwork belongs to
		img image.Image
	}
	tileArtMsg struct {
		id  string // recently-played entry the cover belongs to
		img image.Image
	}
	accountMsg struct {
		acct engine.Account
		err  error
	}
	signedOutMsg        struct{ err error }
	mprisQuitMsg        struct{}
	visualizerOpenedMsg struct {
		service *visualizer.Service
		err     error
		source  string
	}
)

type model struct {
	w, h   int
	t      float64 // seconds since start, drives animation
	phase  phase
	status string
	note   string
	noteAt float64 // m.t when the note was set; expires after 6s
	eng    playback
	cmds   commandSet // optional per-model command factories; zero value uses production defaults
	st     engine.State
	focus  int // 0 queue, 1 recent, 2 player
	selIdx int

	// recently played, shown as a cover grid under the queue
	recent    []engine.Track
	recentSel int
	tileArt   map[string]image.Image // cover per recent entry, nil once fetched and empty

	themeName string            // active theme, persisted across runs
	cfg       map[string]string // parsed config.toml, read once at startup

	statusCh chan string

	badPolls       int  // consecutive failed or signed-out state polls
	mprisListening bool // the quit listener is running; a reconnect must not add a second

	// queue-replacement in flight (slow setQueue on playlists/albums)
	loading         string  // title being loaded, "" = none
	loadSnap        string  // state snapshot at fire time; change = done
	loadStart       float64 // m.t at fire time, for the give-up timeout
	loadHideQueue   bool    // true when the queue itself is being replaced
	initStart       float64 // m.t when the DRM audio initialization began
	initFailed      bool    // a MusicKit error ends this initialization attempt
	initPending     bool    // a local play attempt awaits engine confirmation
	initSeen        bool    // a poll observed Initializing during the pending attempt
	initReadyPolls  int     // consecutive same-snapshot Playing polls while pending
	initTimerActive bool    // initStart is valid and the 30s warning timer is running

	// lyrics for the current track
	ly     lyrics.Lyrics
	lyFor  string // track ID the fetch was fired for
	lyBusy bool

	// album artwork for the current track
	art      image.Image
	artFor   string    // track ID the image belongs to
	artCache *artCache // decoded covers, shared across tracks

	// real spectrum; falls back to fake animation
	vizService  *visualizer.Service
	vizOpening  bool
	vizLive     bool
	vizSource   string
	vizTerminal bool
	vizBands    [32]float64 // smoothed, 0..1
	vizTargets  [32]float64
	vizPeaks    [32]float64 // peak-hold markers, decay slowly
	vizMode     int         // vizBars or vizOrb, persisted across runs
	orbSpin     float64     // torus rotation, accelerated by the beat
	orbWobble   float64     // torus tilt oscillation
	orbKick     float64     // 0..1 beat strength, spikes on a bass hit
	orbKickBase float64     // running average the kick is measured against
	wv          wave        // recorded loudness across the current track

	helpOpen bool // the ? overlay; any key dismisses it

	// EQ panel, swapped in for the visualizer by Q; gains persisted to disk
	eqOpen  bool
	eqGains [10]int
	eqSel   int

	// the a overlay: who is signed in, and the way back out
	acctOpen    bool
	acctLoading bool
	acctBusy    bool // a sign-out is in flight
	acct        engine.Account

	// desktop integration; nil off Linux or without a session bus
	mpris     *mpris.Server
	mprisQuit chan struct{}

	// Last.fm, nil unless configured and authorized
	scrobbler *lastfm.Client
	scrKey    string        // artist+title of the play being timed
	scrStart  time.Time     // wall clock when it started, for the timestamp
	scrPlayed time.Duration // time actually listened, so a seek cannot fake it
	scrDone   bool          // already scrobbled this play

	// search overlay
	searchOpen bool
	sInput     bool
	sQuery     string
	sTab, sSel int
	sBusy      bool
	sRes       engine.SearchResults
}

// captureStatus gives consumers one explicit state without storing another
// copy of the service lifecycle flags. Unavailable retains the existing demo
// fallback, while only live capture may contribute to recorded history.
type captureStatus int

const (
	captureDemo captureStatus = iota
	captureStarting
	captureLive
	captureStale
	captureUnavailable
)

func (m model) captureStatus() captureStatus {
	switch {
	case m.vizLive:
		return captureLive
	case m.vizOpening:
		return captureStarting
	case m.vizService != nil:
		return captureStale
	case m.vizTerminal:
		return captureUnavailable
	default:
		return captureDemo
	}
}

func (m *model) failVisualizer() {
	m.vizService = nil
	m.vizOpening = false
	m.vizLive = false
	m.vizSource = ""
	if !m.vizTerminal {
		m.vizTerminal = true
		m.note = visualizerUnavailableNote
		m.noteAt = m.t
	}
}

func (m *model) closeVisualizerAsync() {
	service := m.vizService
	m.vizService = nil
	m.vizOpening = false
	m.vizLive = false
	m.vizSource = ""
	closeServiceAsync(service)
}

func (m model) audioInitializing() bool {
	return (m.st.Initializing || m.initPending) && !m.initFailed
}

func (m *model) beginAudioInitialization() {
	m.initPending = true
	m.initSeen = false
	m.initReadyPolls = 0
	m.initFailed = false
	m.initStart = m.t
	m.initTimerActive = true
}

func (m *model) resetAudioInitialization() {
	m.initPending = false
	m.initSeen = false
	m.initReadyPolls = 0
	m.initFailed = false
	m.initStart = 0
	m.initTimerActive = false
}

func (m *model) failAudioInitialization() {
	m.initPending = false
	m.initSeen = false
	m.initReadyPolls = 0
	m.initFailed = true
	m.initStart = 0
	m.initTimerActive = false
}

// badPollsToReconnect is 5 polls ≈ 2.5s — long enough that a busy browser or a
// single 10s eval timeout never costs the user a browser restart.
const badPollsToReconnect = 5

// reconnect drops the dead browser and re-runs startup's Connect, which opens a
// visible login window on its own when the session is really gone.
func (m model) reconnect(status string) (tea.Model, tea.Cmd) {
	eng := m.eng
	m.mpris.Close()
	m.mpris, m.eng = nil, nil
	m.badPolls = 0
	m.st = engine.State{}
	m.phase, m.status = phaseBoot, status
	return m, reconnectCmd(eng, m.commands().connect(m.statusCh))
}

func visualizerFrameStale(now, frameAt time.Time) bool {
	return !now.IsZero() && !frameAt.IsZero() &&
		now.Sub(frameAt) > visualizerFrameFreshness
}

func snap(st engine.State) string {
	return fmt.Sprintf("%s|%d|%d", st.Now.ID, len(st.Queue), st.QueuePos)
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.commands().tick(), m.commands().connect(m.statusCh), listenStatus(m.statusCh))
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		if m.focus == focusRecent && !m.recentVisible() {
			m.focus = focusPlayer // the grid just shrank out of the layout
		}
	case tickMsg:
		var visualizerClose tea.Cmd
		now := time.Time(msg)
		m.t += 1.0 / 30
		if m.phase == phaseReady && m.st.Playing && m.st.Pos < m.st.Dur {
			m.st.Pos += time.Second / 30 // optimistic between polls
		}
		if m.loading != "" && m.t-m.loadStart > 12 && !m.audioInitializing() {
			m.loading = "" // give up quietly
		}
		if m.audioInitializing() && m.initTimerActive &&
			m.t-m.initStart > 30 && m.note != audioInitializingWarning {
			m.note, m.noteAt = audioInitializingWarning, m.t
		}
		if m.note != "" && m.t-m.noteAt > 6 &&
			!(m.audioInitializing() && m.note == audioInitializingWarning) {
			m.note = ""
		}
		if m.vizService != nil {
			service := m.vizService
			if service.Err() != nil {
				m.failVisualizer()
				visualizerClose = closeVisualizerCmd(service)
			} else if frame, ok := service.Latest(); ok {
				if visualizerFrameStale(now, frame.At) {
					m.vizTargets = [32]float64{}
					m.vizLive = false
				} else {
					m.vizTargets = frame.Bands
					m.vizLive = frame.Live
					m.vizSource = frame.Source
					m.vizOpening = false
				}
			}
		}
		switch m.captureStatus() {
		case captureLive:
			for i := range m.vizBands {
				m.vizBands[i] += (m.vizTargets[i] - m.vizBands[i]) * 0.45
			}
		case captureDemo, captureUnavailable:
			m.vizBands = simulatedBands(m.t, m.st.Playing)
		default:
			m.vizBands = [32]float64{}
		}
		decayPeaks(&m.vizPeaks, m.vizBands)
		m.orbKickBase, m.orbKick = bassKick(m.orbKickBase, m.orbKick, bassLevel(m.vizBands))
		m.orbSpin, m.orbWobble = orbAdvance(m.orbSpin, m.orbWobble, m.orbKick)
		if m.captureStatus() == captureLive && m.st.Dur > 0 && m.st.Playing {
			m.wv.record(float64(m.st.Pos)/float64(m.st.Dur), bandsLevel(m.vizBands))
		}
		scrobble := m.advanceScrobble(time.Second / 30)
		cmds := []tea.Cmd{m.commands().tick()}
		if visualizerClose != nil {
			cmds = append(cmds, visualizerClose)
		}
		if scrobble != nil {
			cmds = append(cmds, scrobble)
		}
		return m, tea.Batch(cmds...)
	case statusMsg:
		m.status = string(msg)
		return m, listenStatus(m.statusCh)
	case readyMsg:
		m.phase, m.eng = phaseReady, msg.eng
		cmds := []tea.Cmd{
			m.fetchState(),
			m.libraryCmd(), // fills the recently-played grid
		}
		// System audio capture outlives the browser, so a reconnect keeps the
		// visualizer it already has instead of opening a second source.
		if m.vizService == nil {
			m.vizOpening = true
			cmds = append(cmds, m.commands().openVisualizer())
		}
		// A machine with no session bus — a TTY, a container, macOS — simply
		// has no MPRIS, which is not an error worth showing. MPRIS is re-published
		// after a reconnect because its controls hold the old engine.
		if srv, err := m.commands().publishMPRIS(mprisControls{eng: m.eng, quit: m.mprisQuit}); err == nil {
			m.mpris = srv
			if !m.mprisListening {
				m.mprisListening = true
				cmds = append(cmds, listenMPRISQuit(m.mprisQuit))
			}
		}
		return m, tea.Batch(cmds...)
	case mprisQuitMsg:
		m.closeVisualizerAsync()
		m.mpris.Close()
		if m.eng != nil {
			m.eng.Close()
		}
		return m, tea.Quit
	case visualizerOpenedMsg:
		if m.phase != phaseReady || !m.vizOpening {
			if msg.service != nil {
				return m, closeVisualizerCmd(msg.service)
			}
			return m, nil
		}
		if msg.err != nil || msg.service == nil {
			m.failVisualizer()
			return m, nil
		}
		m.vizService = msg.service
		// Opening the backend is not evidence of PCM. The first fresh frame
		// (including real zero-valued silence) ends STARTING on a later tick.
		m.vizLive = false
		m.vizSource = msg.source
	case failMsg:
		m.phase, m.status = phaseFail, msg.err.Error()
	case pollMsg:
		if m.eng == nil {
			return m, nil // a reconnect is in flight; readyMsg restarts the loop
		}
		return m, m.fetchState()
	case stateMsg:
		// A dead session shows up two ways: isAuthorized flips false, or — once
		// the page has navigated away from MusicKit — the poll fails outright.
		// Either way it must persist, so a single hiccup cannot restart us.
		if msg.err != nil || !msg.st.Authed {
			m.badPolls++
			if m.badPolls >= badPollsToReconnect {
				return m.reconnect("session lost — signing in to Apple Music again…")
			}
		} else {
			m.badPolls = 0
		}
		var cmd tea.Cmd
		if msg.err == nil {
			wasInitializing := m.st.Initializing
			newSt := msg.st
			// keep the optimistic clock for small drift: the polled position
			// lags by up to the poll interval and would flick lyrics back
			if newSt.Now.ID == m.st.Now.ID {
				if d := newSt.Pos - m.st.Pos; d > -1500*time.Millisecond && d < 1500*time.Millisecond {
					newSt.Pos = m.st.Pos
				}
			}
			m.st = newSt
			m.mpris.Update(mprisState(m.st))
			nowInitializing := m.st.Initializing
			if m.initPending && nowInitializing {
				m.initSeen = true
				m.initReadyPolls = 0
			}
			if nowInitializing && !m.initTimerActive {
				m.initStart = m.t
				m.initTimerActive = true
			}
			if m.initPending && !nowInitializing {
				complete := false
				if m.st.Playing {
					complete = m.initSeen || snap(m.st) != m.loadSnap
					if !complete {
						m.initReadyPolls++
						complete = m.initReadyPolls >= 2
					}
				} else {
					m.initReadyPolls = 0
				}
				if complete {
					m.resetAudioInitialization()
					m.loading = ""
				}
			}
			m.selIdx = min(m.selIdx, max(0, len(m.st.Queue)-1))
			if m.st.Err != "" {
				m.note, m.noteAt = m.st.Err, m.t // MusicKit rejection from an action
				m.failAudioInitialization()
				m.loading = ""
			} else if wasInitializing && !nowInitializing {
				m.initFailed = false
				if !m.initPending {
					m.initStart = 0
					m.initTimerActive = false
				}
			}
			if m.loading != "" && snap(m.st) != m.loadSnap && !m.audioInitializing() {
				m.loading = "" // queue switched — loading done
			}
			if id := m.st.Now.ID; id != "" && id != m.lyFor {
				m.lyFor, m.lyBusy, m.ly = id, true, lyrics.Lyrics{}
				cmd = m.commands().lyrics(id, m.st.Now.Artist, m.st.Now.Title, m.st.Now.Duration)
			}
			if id := m.st.Now.ID; id != "" && id != m.artFor {
				m.wv.reset()
				m.artFor, m.art = id, nil
				if cached, ok := m.artCache.get(id); ok {
					m.art = cached
					m.applyAutoTheme()
				} else if tmpl := m.st.Now.Art; tmpl != "" {
					cmd = tea.Batch(cmd, m.commands().artwork(id, tmpl))
				}
			}
			// A non-flat EQ that is not yet attached to the current <audio>
			// element covers first playback, element swaps and page
			// reloads/reconnects with one mechanism: just keep re-asking.
			if m.st.Now.ID != "" && !m.st.EQ && !eqIsFlat(m.eqGains) {
				eng, gains := m.eng, m.eqGains
				cmd = tea.Batch(cmd, doCmd(func() error { return eng.SetEQ(gains) }))
			}
		}
		poll := m.commands().poll()
		return m, tea.Batch(poll, cmd)
	case lyricsMsg:
		if msg.id == m.lyFor {
			m.ly, m.lyBusy = msg.ly, false
		}
	case artMsg:
		if msg.id == m.artFor && msg.img != nil {
			m.art = msg.img
			m.artCache.put(msg.id, msg.img)
			m.applyAutoTheme()
		}
	case noteMsg:
		m.note, m.noteAt = string(msg), m.t
		m.loading = "" // action failed — stop the spinner
		if m.st.Initializing || m.initPending {
			m.failAudioInitialization()
		}
	case searchMsg:
		m.sBusy = false
		if msg.err != nil {
			m.note, m.noteAt = msg.err.Error(), m.t
		}
		// Apple grants the library and the listening history as separate
		// privileges, so one can be refused while the other answers — whatever
		// did arrive is shown alongside the reason rather than dropped with it.
		m.sRes, m.sSel, m.sInput = msg.res, 0, msg.keepInput
		if !msg.keepInput && m.sTab == 0 {
			m.sTab = 1 // catalog search has no RECENT — land on SONGS
		}
		// Only a library load carries recently played; a catalog search must
		// not wipe the grid.
		if len(msg.res.Recent) > 0 {
			m.recent = msg.res.Recent
			m.recentSel = min(m.recentSel, len(m.recent)-1)
			return m, tea.Batch(m.fetchTileArtCmds()...)
		}
	case tileArtMsg:
		if m.tileArt == nil {
			m.tileArt = map[string]image.Image{}
		}
		m.tileArt[msg.id] = msg.img // nil is a valid "tried and failed" marker
	case accountMsg:
		m.acctLoading = false
		if msg.err == nil {
			m.acct = msg.acct
		}
		// A failed lookup is not worth a banner: the overlay still shows
		// "signed in" and the one thing it is there for still works.
	case signedOutMsg:
		m.acctOpen, m.acctBusy, m.acct = false, false, engine.Account{}
		if msg.err != nil {
			m.note, m.noteAt = "sign out failed: "+msg.err.Error(), m.t
			return m, nil
		}
		return m.reconnect("signed out — sign in to Apple Music")
	case tea.MouseMsg:
		return m.updateMouse(msg)
	case tea.KeyMsg:
		// Any key dismisses help, and that key does nothing else — reading the
		// list should never fire the command you were looking up.
		if m.helpOpen {
			m.helpOpen = false
			return m, nil
		}
		if m.acctOpen {
			return m.updateAccount(msg)
		}
		if m.searchOpen {
			return m.updateSearch(msg)
		}
		return m.updateKeys(msg)
	}
	return m, nil
}

// Focus zones, cycled by Tab in visual order down the left column and then to
// the transport bar.
const (
	focusQueue = iota
	focusRecent
	focusPlayer
)

// recentCols is how many cover tiles sit in a grid row. Fixed at two: it keeps
// the tiles readable at the width the left column actually gets.
const recentCols = 2

func (m model) updateKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.phase != phaseReady {
		if s := msg.String(); s == "q" || s == "ctrl+c" {
			m.closeVisualizerAsync()
			return m, tea.Quit
		}
		return m, nil
	}
	if m.audioInitializing() {
		switch msg.String() {
		case " ", "n", "p", "enter":
			m.note, m.noteAt = audioInitializingText, m.t
			return m, nil
		}
	}
	eng := m.eng
	if m.eqOpen {
		switch msg.String() {
		case "esc", "Q":
			m.eqOpen = false
			return m, nil
		case "left", "h":
			m.eqSel = (m.eqSel + 9) % 10
			return m, nil
		case "right", "l":
			m.eqSel = (m.eqSel + 1) % 10
			return m, nil
		case "down", "j":
			m.eqGains[m.eqSel] = clampEQGain(m.eqGains[m.eqSel] - 1)
			saveEQGains(m.eqGains)
			gains := m.eqGains
			return m, doCmd(func() error { return eng.SetEQ(gains) })
		case "up", "k":
			m.eqGains[m.eqSel] = clampEQGain(m.eqGains[m.eqSel] + 1)
			saveEQGains(m.eqGains)
			gains := m.eqGains
			return m, doCmd(func() error { return eng.SetEQ(gains) })
		case "P":
			m.eqGains = nextEQPreset(m.eqGains)
			saveEQGains(m.eqGains)
			gains := m.eqGains
			return m, doCmd(func() error { return eng.SetEQ(gains) })
		case "0":
			m.eqGains = [10]int{}
			saveEQGains(m.eqGains)
			gains := m.eqGains
			return m, doCmd(func() error { return eng.SetEQ(gains) })
		}
	}
	switch msg.String() {
	case "q", "ctrl+c":
		m.closeVisualizerAsync()
		m.mpris.Close()
		if eng != nil {
			eng.Close()
		}
		return m, tea.Quit
	case "tab":
		m.focus = (m.focus + 1) % 3
		if m.focus == focusRecent && !m.recentVisible() {
			m.focus = focusPlayer
		}
	case " ":
		m.st.Playing = !m.st.Playing
		return m, doCmd(func() error { return eng.PlayPause() })
	case "n":
		return m, doCmd(func() error { return eng.Next() })
	case "p":
		return m, doCmd(func() error { return eng.Prev() })
	case "s":
		return m, doCmd(func() error { return eng.ToggleShuffle() })
	case "r":
		return m, doCmd(func() error { return eng.CycleRepeat() })
	case "?":
		m.helpOpen = true
	case "a":
		m.acctOpen, m.acctLoading = true, true
		return m, accountCmd(eng)
	case "v":
		m.vizMode = (m.vizMode + 1) % vizModes
		saveVizMode(m.vizMode)
		m.note, m.noteAt = "visualizer · "+vizModeName(m.vizMode), m.t
	case "Q":
		m.eqOpen = true
	case "t":
		next := nextTheme(m.themeName)
		m.themeName = next.name
		applyTheme(next)
		saveThemeName(next.name)
		m.applyAutoTheme()
		m.note, m.noteAt = "theme · "+next.name, m.t
	case "R": // reload the web player when it wedges
		m.note, m.noteAt = "reloading engine…", m.t
		m.lyFor = ""
		m.resetAudioInitialization()
		return m, doCmd(func() error { return eng.Reload() })
	case "/":
		m.searchOpen, m.sInput = true, true
		if strings.TrimSpace(m.sQuery) == "" {
			m.sBusy, m.sTab, m.sSel = true, 0, 0 // open on recently played
			return m, m.libraryCmd()
		}
	case "j", "down":
		switch m.focus {
		case focusQueue:
			m.selIdx = min(m.selIdx+1, max(0, len(m.st.Queue)-1))
		case focusRecent:
			m.recentSel = min(m.recentSel+recentCols, max(0, len(m.recent)-1))
		default:
			m.st.Volume = max(m.st.Volume-5, 0)
			v := m.st.Volume
			return m, doCmd(func() error { return eng.SetVolume(v) })
		}
	case "k", "up":
		switch m.focus {
		case focusQueue:
			m.selIdx = max(m.selIdx-1, 0)
		case focusRecent:
			m.recentSel = max(m.recentSel-recentCols, 0)
		default:
			m.st.Volume = min(m.st.Volume+5, 100)
			v := m.st.Volume
			return m, doCmd(func() error { return eng.SetVolume(v) })
		}
	case "enter":
		if m.focus == focusQueue && len(m.st.Queue) > 0 {
			i := m.selIdx
			m.loading = m.st.Queue[i].Title
			m.loadSnap, m.loadStart, m.loadHideQueue = snap(m.st), m.t, false
			return m, doCmd(func() error { return eng.JumpTo(i) })
		}
		if m.focus == focusRecent && m.recentSel < len(m.recent) {
			tr := m.recent[m.recentSel]
			m.loading, m.loadSnap, m.loadStart = tr.Title, snap(m.st), m.t
			m.loadHideQueue = true // an album or playlist replaces the whole queue
			m.beginAudioInitialization()
			kind, id := tr.Kind, tr.ID
			return m, doCmd(func() error { return eng.Play(kind, id) })
		}
	case "left":
		if m.focus == focusRecent {
			m.recentSel = max(m.recentSel-1, 0)
			return m, nil
		}
		if m.focus == focusPlayer {
			m.st.Pos = max(m.st.Pos-5*time.Second, 0)
			m.wv.reset()
			p := m.st.Pos
			return m, doCmd(func() error { return eng.SeekTo(p) })
		}
	case "right":
		if m.focus == focusRecent {
			m.recentSel = min(m.recentSel+1, max(0, len(m.recent)-1))
			return m, nil
		}
		if m.focus == focusPlayer {
			m.st.Pos = min(m.st.Pos+5*time.Second, m.st.Dur)
			m.wv.reset()
			p := m.st.Pos
			return m, doCmd(func() error { return eng.SeekTo(p) })
		}
	}
	return m, nil
}
