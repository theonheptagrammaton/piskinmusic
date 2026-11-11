package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Clicking only works if the hit-test agrees with the renderer down to the
// cell, so the panel arithmetic lives here once and View reads it rather than
// recomputing it. Same for the progress bar: transportPanel takes its width
// from progressGeometry, so a click cannot land where the bar is not drawn.

// rect uses terminal cells and half-open bounds. Empty rectangles cannot be hit.
type rect struct{ x, y, w, h int }

func (r rect) contains(x, y int) bool {
	return x >= r.x && y >= r.y && x < r.x+r.w && y < r.y+r.h
}

func (r rect) clipped(w, h int) rect {
	x, y := max(0, r.x), max(0, r.y)
	right, bottom := min(max(0, w), r.x+r.w), min(max(0, h), r.y+r.h)
	if right <= x || bottom <= y {
		return rect{}
	}
	return rect{x, y, right - x, bottom - y}
}

func clipTargets(targets []keyTarget, w, h int) []keyTarget {
	out := targets[:0]
	for _, a := range targets {
		a.rect = a.rect.clipped(w, h)
		if a.w > 0 && a.h > 0 {
			out = append(out, a)
		}
	}
	return out
}

type layout struct {
	visible                                                bool
	queueBody, recentBody, transport, progress, play       rect
	main, libraryBody                                      rect
	library, queue, nowPlaying, visualizer, lyrics, recent rect
	nav                                                    [4]rect
	actions                                                []keyTarget
	libraryTabs                                            [3]rect

	recentGrid  recentGridLayout
	recentTiles [recentCols]rect

	topH   int // rows above the transport
	leftW  int // full width of the queue column, borders included
	rightW int
	qw, qh int // queue content width and full panel height
	recH   int // full height of the recently-played box, 0 when hidden
}

func (m model) layout() layout {
	key := layoutKey{m.w, m.h, m.screen, m.queueOpen, m.eqOpen, len(m.recent) > 0}
	var l layout
	if m.renderCache != nil && m.renderCache.hasGeometry && m.renderCache.geometryKey == key {
		l = m.renderCache.geometry
	} else {
		l = m.staticLayout()
		if m.renderCache != nil {
			m.renderCache.geometry, m.renderCache.geometryKey, m.renderCache.hasGeometry = l, key, true
		}
	}
	if l.visible {
		// Clock width, shuffle, repeat and volume affect transport geometry.
		barX, barW := m.progressGeometry(l.transport.w)
		l.progress = rect{l.transport.x + barX, l.transport.y + 1, barW, 1}
	}
	return l
}

func (m model) staticLayout() layout {
	l := layout{visible: m.w >= 40 && m.h >= 12, topH: m.h - 4}
	if !l.visible {
		return l
	}
	x := 1
	for i := range screenNames {
		label := m.screenLabel(i)
		l.nav[i] = rect{x, 1, lipgloss.Width(label), 1}
		x += lipgloss.Width(label) + 2
	}
	labels := []string{"/ Search", "c Queue", "Q EQ", "a Account", "? Help"}
	toolbarWidth := func(labels []string) int {
		width := len(labels) - 1
		for _, label := range labels {
			width += lipgloss.Width(label)
		}
		return width
	}
	toolbarX := 2 + lipgloss.Width("PISKIN MUSIC")
	if m.w < toolbarX+toolbarWidth(labels)+1 {
		labels = []string{"/ Find", "c Queue", "Q EQ", "a", "?"}
	}
	keys := []string{"/", "c", "Q", "a", "?"}
	x = max(toolbarX, m.w-1-toolbarWidth(labels))
	for i, label := range labels {
		l.actions = append(l.actions, keyTarget{rect{x, 0, lipgloss.Width(label), 1}, keys[i], label})
		x += lipgloss.Width(label) + 1
	}
	bodyH := m.h - 6
	l.main = rect{0, 2, m.w, bodyH}
	if m.queueOpen && !m.eqOpen {
		l.queue = l.main
	} else if !m.eqOpen {
		// Keep all five listening regions visible, even at 40×12. Narrow
		// windows stack three compact panels beside the two browsing lists.
		l.leftW = min(38, max(16, m.w/3))
		l.rightW = m.w - l.leftW
		libraryH := max(3, bodyH/2)
		if m.w >= 100 && bodyH >= 29 && len(m.recent) > 0 {
			l.recH = min(10, bodyH/3)
			libraryH = max(3, (bodyH-l.recH)/2)
			l.recent = rect{0, 2 + bodyH - l.recH, l.leftW, l.recH}
			l.recentGrid = recentGrid(l.leftW-2, l.recH-1)
			if l.recentGrid.artCols == 0 {
				l.recH, l.recent = 0, rect{}
				libraryH = bodyH / 2
			}
		}
		l.library = rect{0, 2, l.leftW, libraryH}
		l.queue = rect{0, 2 + libraryH, l.leftW, bodyH - libraryH - l.recH}
		if m.w >= 100 {
			middleW := l.rightW * 55 / 100
			l.nowPlaying = rect{l.leftW, 2, middleW, bodyH / 2}
			l.visualizer = rect{l.leftW, 2 + bodyH/2, middleW, bodyH - bodyH/2}
			l.lyrics = rect{l.leftW + middleW, 2, l.rightW - middleW, bodyH}
		} else {
			coverH, vizH := max(2, bodyH/3), max(2, bodyH/3)
			l.nowPlaying = rect{l.leftW, 2, l.rightW, coverH}
			l.visualizer = rect{l.leftW, 2 + coverH, l.rightW, vizH}
			l.lyrics = rect{l.leftW, 2 + coverH + vizH, l.rightW, bodyH - coverH - vizH}
		}
		l.libraryBody = rect{1, l.library.y + 2, l.library.w - 2, l.library.h - 2}
		x = 1
		for i := range libraryNames {
			label := libraryTabLabel(l.library.w-2, i)
			l.libraryTabs[i] = rect{x, l.library.y + 1, lipgloss.Width(label) + 2, 1}
			x += lipgloss.Width(label) + 3
		}
		if l.recH > 0 {
			g := l.recentGrid
			l.recentBody = rect{1, l.recent.y + 1, l.recent.w - 2, l.recent.h - 1}
			for c := range l.recentTiles {
				l.recentTiles[c] = rect{1 + g.gap*(c+1) + g.artCols*c, l.recentBody.y, g.artCols, g.artRows + 2}
			}
		}
	}
	if l.queue.h > 0 {
		l.qw, l.qh = l.queue.w-2, l.queue.h
		l.queueBody = rect{l.queue.x + 1, l.queue.y + 1, l.qw, l.qh - 1}
	}
	l.transport = rect{1, l.topH + 1, m.w - 2, 2}
	playX := 4
	if l.transport.w < 68 {
		playX = 3
	}
	l.play = rect{l.transport.x + playX, l.transport.y + 1, 1, 1}
	return l
}

// queueRows is how many tracks the queue panel shows at once, and queueTop the
// screen row the first of them is drawn on, after the single heading row.
func (l layout) queueRows() int { return l.queueBody.h }
func (l layout) queueTop() int  { return l.queueBody.y }

// recentTop is the screen row the cover grid starts on, directly below the
// queue box.
func (l layout) recentTop() int { return l.recentBody.y }

// progressGeometry returns the column the progress bar starts at within the
// transport's content, and its width. The wide transport reserves ten cells
// for previous/play/next; the compact form uses four for play.
func (m model) progressGeometry(w int) (x, width int) {
	if w < 68 {
		x = 4 + lipgloss.Width(fmtTime(m.st.Pos)+" ")
		return x, max(1, w-x-lipgloss.Width(" "+fmtTime(m.st.Dur)))
	}
	times := fmtTime(m.st.Pos) + " "
	timee := " " + fmtTime(m.st.Dur)
	shuf, rep := "off", [3]string{"off", "one", "all"}[min(m.st.Repeat, 2)]
	if m.st.Shuffle {
		shuf = "on"
	}
	right := fmt.Sprintf("⇄ %s  ↻ %s  vol ", shuf, rep) +
		strings.Repeat("▮", 6) + fmt.Sprintf(" %d%% ", m.st.Volume)
	barW := max(10, w-lipgloss.Width(times)-lipgloss.Width(timee)-lipgloss.Width(right)-12)
	return 10 + lipgloss.Width(times), barW
}

// recentGrid is shared by the component and tile hit testing, including the
// width limit, caption rows and gaps between covers.
type recentGridLayout struct{ artCols, artRows, gap int }

func recentGrid(w, rows int) recentGridLayout {
	const minGap = 2
	artRows := rows - 2
	artCols := artRows * 2
	if maxCols := (w - (recentCols+1)*minGap) / recentCols; artCols > maxCols {
		artCols = maxCols - maxCols%2
		artRows = artCols / 2
	}
	if artCols < 6 || artRows < 2 {
		return recentGridLayout{}
	}
	return recentGridLayout{artCols, artRows, (w - recentCols*artCols) / (recentCols + 1)}
}

func (l layout) recentTileAt(x, y, selected, count int) int {
	start := selected / recentCols * recentCols
	for c, tile := range l.recentTiles {
		if tile.contains(x, y) && start+c < count {
			return start + c
		}
	}
	return -1
}

// recentVisible reports whether the grid is on screen and therefore focusable.
func (m model) recentVisible() bool {
	return m.layout().recH > 0
}
