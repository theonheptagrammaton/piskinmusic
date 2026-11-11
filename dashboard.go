package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Dashboard panels reserve just one heading row. At the minimum size this
// leaves real content in every region instead of hiding a listening panel.
func dashboardPanel(title, body string, r rect, focused bool) string {
	w := r.w - 2
	if focused {
		title = "[" + title + "]"
	}
	head := brandText(ansi.Truncate(title, w, ""))
	if room := w - lipgloss.Width(head) - 1; room > 0 {
		head += " " + lipgloss.NewStyle().Foreground(borderDim).Render(strings.Repeat("─", room))
	}
	return lipgloss.NewStyle().Padding(0, 1).Render(pad(head, w) + "\n" + fitBlock(body, w, r.h-1))
}

func (m model) dashboard(l layout) string {
	rows := make([]string, l.main.h)
	for i := range rows {
		rows[i] = strings.Repeat(" ", m.w)
	}
	place := func(r rect, title, body string, focused bool) {
		if r.w <= 0 || r.h <= 0 {
			return
		}
		lines := strings.Split(dashboardPanel(title, body, r, focused), "\n")
		for i := 0; i < r.h; i++ {
			y := r.y - l.main.y + i
			rows[y] = ansi.Cut(rows[y], 0, r.x) + pad(lines[i], r.w) + ansi.Cut(rows[y], r.x+r.w, m.w)
		}
	}
	if m.eqOpen {
		return strings.Join(rows, "\n")
	}
	queueTitle := m.queueTitle()
	if m.queueOpen {
		queueTitle += " · c / esc close"
	}
	place(l.queue, queueTitle, m.queuePanel(l.qw, l.qh), m.focus == focusQueue)
	if m.queueOpen {
		return strings.Join(rows, "\n")
	}
	place(l.library, "LIBRARY · "+strings.ToLower(libraryNames[m.libraryTab]),
		m.libraryPanel(l.library.w-2, l.library.h), m.focus == focusLibrary)
	place(l.nowPlaying, "NOW PLAYING", m.dashboardCover(l.nowPlaying.w-2, l.nowPlaying.h-1), m.screen == screenFocus)
	vizTitle := m.visualizerTitle()
	// Keep capture identity visible before the mode when a title must shrink.
	if l.visualizer.w < 30 {
		vizTitle = strings.Replace(vizTitle, "VISUALIZER", "VIZ", 1)
		if m.captureStatus() == captureDemo {
			vizTitle = "VIZ DEMO · SIMULATED"
		}
	}
	place(l.visualizer, vizTitle+" · "+vizModeName(m.vizMode),
		m.vizPanel(l.visualizer.w-2, l.visualizer.h), m.screen == screenVisualizer)
	place(l.lyrics, "LYRICS", m.lyricsText(l.lyrics.w-2, l.lyrics.h-1), m.screen == screenLyrics)
	if l.recH > 0 {
		place(l.recent, m.recentTitle(), m.renderRecentPanel(l.recent.w-2, l.recent.h, l.recentGrid), m.focus == focusRecent)
	}
	return strings.Join(rows, "\n")
}

func (m model) dashboardCover(w, rows int) string {
	if w <= 0 || rows <= 0 {
		return ""
	}
	if !configBool(m.cfg, "artwork.enabled", true) {
		return centeredBlock(m.metadata(w, rows), w, rows)
	}
	cols := min(48, min(rows*2, w*45/100))
	cols -= cols % 2
	artRows := cols / 2
	if cols < 2 {
		return fitBlock(m.metadata(w, rows), w, rows)
	}
	cover := placeholderTile(cols, artRows)
	if m.art != nil {
		cover = m.artworkRows(m.art, cols, artRows)
	}
	art := strings.Join(cover, "\n")
	textW := w - cols - 2
	text := fitBlock(m.metadata(textW, rows), textW, min(rows, 8))
	return centeredBlock(lipgloss.JoinHorizontal(lipgloss.Center, art, "  ", text), w, rows)
}
