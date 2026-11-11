package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/k1y0miiii/applemusic-tui/engine"
)

type screen int

const (
	screenLibrary screen = iota
	screenFocus
	screenVisualizer
	screenLyrics
)

var screenNames = [...]string{"Library", "Cover", "Visualizer", "Lyrics"}
var libraryNames = [...]string{"Albums", "Playlists", "Recent"}

// screenLabel carries a textual active marker even in terminals without color.
func (m model) screenLabel(i int) string {
	label := fmt.Sprintf("%d %s", i+1, screenNames[i])
	if m.w < 50 && i == 2 {
		label = "3 Viz"
	}
	if i == int(m.screen) {
		label = "[" + label + "]"
	}
	return label
}

type keyTarget struct {
	rect
	key, label string
}

func (m *model) selectScreen(s screen) {
	m.screenFocus[m.screen] = m.focus
	m.screen = s
	m.focus = m.screenFocus[s]
	m.normalizeFocus()
}

func (m model) focusZones() []int {
	l := m.layout()
	var zones []int
	if l.queueBody.h > 0 {
		zones = append(zones, focusQueue)
	}
	if l.recH > 0 {
		zones = append(zones, focusRecent)
	}
	if l.libraryBody.h > 0 {
		zones = append(zones, focusLibrary)
	}
	return append(zones, focusPlayer)
}
func (m *model) normalizeFocus() {
	zones := m.focusZones()
	for _, f := range zones {
		if f == m.focus {
			return
		}
	}
	m.focus = focusPlayer
}
func (m *model) cycleFocus(reverse bool) {
	zones := m.focusZones()
	for i, f := range zones {
		if f == m.focus {
			step := 1
			if reverse {
				step = len(zones) - 1
			}
			m.focus = zones[(i+step)%len(zones)]
			return
		}
	}
	m.focus = zones[0]
}
func (m *model) toggleQueue() {
	m.queueOpen = !m.queueOpen
	if m.queueOpen {
		m.returnFocus, m.focus = m.focus, focusQueue
	} else {
		m.focus = m.returnFocus
		m.normalizeFocus()
	}
}
func (m model) libraryList(tab int) []engine.Track {
	switch tab {
	case 0:
		return m.library.Albums
	case 1:
		return m.library.Playlists
	default:
		return m.recent
	}
}
func (m model) playLibrarySelection() (tea.Model, tea.Cmd) {
	list := m.libraryList(m.libraryTab)
	i := m.librarySel[m.libraryTab]
	if i < 0 || i >= len(list) {
		return m, nil
	}
	if m.audioInitializing() {
		m.note, m.noteAt = audioInitializingText, m.t
		return m, nil
	}
	tr := list[i]
	m.loading, m.loadSnap, m.loadStart, m.loadHideQueue = tr.Title, snap(m.st), m.t, true
	m.beginAudioInitialization()
	eng := m.eng
	return m, doCmd(func() error { return eng.Play(tr.Kind, tr.ID) })
}

// listWindow is shared by rendering and pointer selection, including the final page.
func listWindow(selected, count, rows int) int { return max(0, min(selected-rows/2, count-rows)) }

// Callers supply a marker for reduced-color terminals.
func selectedRow(text string, w int) string {
	if rainbow {
		return lipgloss.NewStyle().Background(selBg).Bold(true).Render(pad(brandText(strings.TrimRight(text, " ")), w))
	}
	return lipgloss.NewStyle().Foreground(selectedText).Background(selBg).Bold(true).Render(pad(text, w))
}

func libraryTabLabel(w, i int) string {
	if w < 31 {
		return []string{"A", "P", "R"}[i]
	}
	return libraryNames[i]
}

func (m model) libraryPanel(w, h int) string {
	tabs := ""
	for i := range libraryNames {
		name := libraryTabLabel(w, i)
		label := " " + name + " "
		if i == m.libraryTab {
			label = "[" + name + "]"
		}
		st := lipgloss.NewStyle().Foreground(fgFaint)
		if i == m.libraryTab {
			st = st.Foreground(fgBright).Bold(true)
		}
		tabs += st.Render(label) + " "
	}
	categoryHint := lipgloss.NewStyle().Foreground(fgFaint).Render("[ / ] category")
	if gap := w - lipgloss.Width(tabs) - lipgloss.Width(categoryHint); gap >= 2 {
		tabs += strings.Repeat(" ", gap) + categoryHint
	}
	rows := h - 2
	list := m.libraryList(m.libraryTab)
	sel := m.librarySel[m.libraryTab]
	start := listWindow(sel, len(list), rows)
	lines := []string{pad(tabs, w)}
	for r := 0; r < rows; r++ {
		i := start + r
		line := ""
		if i < len(list) {
			marker := "  "
			if i == sel {
				marker = "▸ "
			}
			line = listTrack(marker, list[i].Title, list[i].Artist, "", w, m.focus == focusLibrary && i == sel, false)
		} else if r == 0 && len(list) == 0 {
			switch {
			case m.libraryBusy:
				line = " loading library…"
			case m.libraryError != "":
				line = " library unavailable · / retry"
			default:
				line = " no " + strings.ToLower(libraryNames[m.libraryTab]) + " · / search"
			}
		}
		lines = append(lines, pad(line, w))
	}
	return strings.Join(lines, "\n")
}
func (m model) metadata(w, rows int) string {
	title, artist, album := m.st.Now.Title, m.st.Now.Artist, m.st.Now.Album
	if title == "" {
		title = "Nothing playing"
		artist = "/ Search Apple Music"
	}
	if w <= 0 || rows <= 0 {
		return ""
	}
	wrap := func(s string, tone lipgloss.Color, bold bool) string {
		lines := strings.Split(ansi.Wrap(s, w, ""), "\n")
		if len(lines) > 2 {
			lines = []string{lines[0], strings.TrimRight(ellipsize(strings.Join(lines[1:], " "), w), " ")}
		}
		return lipgloss.NewStyle().Foreground(tone).Bold(bold).Render(strings.Join(lines, "\n"))
	}
	lines := []string{wrap(title, fgBright, true)}
	if rows >= 8 {
		lines = append(lines, "")
	}
	lines = append(lines, wrap(artist, fgDim, false))
	if album != "" {
		lines = append(lines, wrap(album, fgFaint, false))
	}
	if rows >= 10 {
		state := "PAUSED"
		if title == "Nothing playing" {
			state = "READY TO LISTEN"
		} else if m.st.Playing {
			state = "PLAYING"
		}
		lines = append(lines, "", lipgloss.NewStyle().Foreground(fgFaint).Render(state))
	}
	if m.audioInitializing() {
		lines = append(lines, ansi.Wrap(m.audioInitializingLine(), w, ""))
	}
	if m.note != "" {
		lines = append(lines, wrap(m.note, fgDim, false))
	}
	return strings.Join(lines, "\n")
}
func (m model) focusPanel(w, h int) string {
	rows := h - 1
	if w <= 0 || rows <= 0 {
		return ""
	}
	const gap = 5
	cols := min(48, min((rows-2)*2, w-gap-24))
	cols -= cols % 2
	if !configBool(m.cfg, "artwork.enabled", true) || m.art == nil || cols < artworkMinCols || rows < 8 {
		textW := min(w, 50)
		return centeredBlock(m.metadata(textW, rows), w, rows)
	}
	art := strings.Join(m.artworkRows(m.art, cols, cols/2), "\n")
	textW := min(40, w-cols-gap)
	text := m.metadata(textW, rows)
	composition := lipgloss.JoinHorizontal(lipgloss.Center, art, strings.Repeat(" ", gap), fitBlock(text, textW, min(rows, lipgloss.Height(text))))
	return centeredBlock(composition, w, rows)
}

// Center a natural-size composition; clamp before placing so long metadata
// cannot push content or pointer geometry beyond the terminal.
func centeredBlock(s string, w, h int) string {
	if w <= 0 || h <= 0 {
		return ""
	}
	s = fitBlock(s, min(w, lipgloss.Width(s)), min(h, lipgloss.Height(s)))
	return fitBlock(lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, s), w, h)
}

func (m model) header(l layout) string {
	key := headerKey{w: m.w, screen: m.screen, bright: fgBright, dim: fgDim,
		selected: selectedText, selection: selBg, accent: accentHi, rainbow: rainbow, profile: lipgloss.ColorProfile()}
	if m.renderCache != nil && m.renderCache.header != "" && m.renderCache.headerKey == key {
		return m.renderCache.header
	}
	brand := brandText("PISKIN MUSIC")
	top := " " + brand
	if len(l.actions) > 0 {
		top += strings.Repeat(" ", max(1, l.actions[0].x-lipgloss.Width(top)))
		for _, a := range l.actions {
			top += lipgloss.NewStyle().Foreground(fgDim).Render(a.label) + " "
		}
	}
	nav := " "
	for i, r := range l.nav {
		label := m.screenLabel(i)
		if i == int(m.screen) {
			label = selectedRow(label, r.w)
		} else {
			label = lipgloss.NewStyle().Foreground(fgDim).Render(label)
		}
		nav += label + "  "
	}
	result := pad(top, m.w) + "\n" + pad(nav, m.w)
	if m.renderCache != nil {
		m.renderCache.header, m.renderCache.headerKey = result, key
	}
	return result
}

// fitBlock clips ANSI and Unicode by terminal cells. Every frame has bounded
// dimensions; callers retain ownership of hit geometry and scrolling.
func fitBlock(s string, w, h int) string {
	if w <= 0 || h <= 0 {
		return ""
	}
	lines := strings.Split(s, "\n")
	out := make([]string, h)
	for i := range out {
		if i < len(lines) {
			out[i] = pad(lines[i], w)
		} else {
			out[i] = strings.Repeat(" ", w)
		}
	}
	return strings.Join(out, "\n")
}

func (m model) compactTransport(w, barW int) string {
	title := m.st.Now.Title
	if title == "" {
		title = "Nothing playing · / search"
	}
	if m.audioInitializing() {
		title = audioInitializingText
	} else if m.loading != "" {
		title = "loading " + m.loading
	} else if m.note != "" {
		title = m.note
	}
	icon := "▶"
	if m.st.Playing {
		icon = "⏸"
	}
	frac := 0.0
	if m.st.Dur > 0 {
		frac = min(1, max(0, float64(m.st.Pos)/float64(m.st.Dur)))
	}
	line := "   " + lipgloss.NewStyle().Foreground(accentHi).Render(icon) +
		lipgloss.NewStyle().Foreground(fgDim).Render(fmtTime(m.st.Pos)+" ") +
		m.progressBar(barW, int(frac*float64(barW))) +
		lipgloss.NewStyle().Foreground(fgDim).Render(" "+fmtTime(m.st.Dur))
	mark := "  "
	if m.focus == focusPlayer {
		mark = "▸ "
	}
	return lipgloss.NewStyle().Foreground(fgBright).Bold(true).Render(ellipsize(mark+title, w)) + "\n" + pad(line, w)
}

func (m model) eqActions(l layout) []keyTarget {
	x := l.main.x + 1
	var out []keyTarget
	for _, a := range []struct{ key, label string }{{"down", "−"}, {"up", "+"}, {"P", "P Preset"}, {"0", "0 Flat"}, {"Q", "× Close"}} {
		out = append(out, keyTarget{rect{x, l.main.y + l.main.h - 2, lipgloss.Width(a.label), 1}, a.key, a.label})
		x += lipgloss.Width(a.label) + 1
	}
	return clipTargets(out, m.w, m.h)
}
func (m model) eqBody(w, h int, l layout) string {
	footer := ""
	for _, a := range m.eqActions(l) {
		footer += a.label + " "
	}
	return eqPanel(w, h-1, m.eqGains, m.eqSel) + "\n" + pad(footer, w)
}
func (m model) transportActions(l layout) []keyTarget {
	if l.transport.w < 68 {
		return nil
	}
	shuf, rep := "off", [3]string{"off", "one", "all"}[min(m.st.Repeat, 2)]
	if m.st.Shuffle {
		shuf = "on"
	}
	right := fmt.Sprintf("⇄ %s  ↻ %s  vol ", shuf, rep) + strings.Repeat("▮", 6) + fmt.Sprintf(" %d%% ", m.st.Volume)
	x := l.transport.x + l.transport.w - lipgloss.Width(right)
	return []keyTarget{
		{rect{l.transport.x + 1, l.transport.y + 1, 1, 1}, "p", "‹"},
		{rect{l.transport.x + 7, l.transport.y + 1, 1, 1}, "n", "›"},
		{rect{x, l.transport.y + 1, lipgloss.Width("⇄ " + shuf), 1}, "s", ""},
		{rect{x + lipgloss.Width("⇄ "+shuf+"  "), l.transport.y + 1, lipgloss.Width("↻ " + rep), 1}, "r", ""},
	}
}
