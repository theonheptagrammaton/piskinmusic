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
	visible                                          bool
	queueBody, recentBody, transport, progress, play rect
	main, libraryBody                                rect
	nav                                              [4]rect
	actions                                          []keyTarget
	libraryTabs                                      [3]rect
	queueY                                           int

	recentGrid  recentGridLayout
	recentTiles [recentCols]rect

	topH   int // rows above the transport
	leftW  int // full width of the queue column, borders included
	rightW int
	qw, qh int // queue panel content size
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
	l := layout{visible: m.w >= 40 && m.h >= 12, topH: m.h - 4, queueY: 2}
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
	showQueue := m.queueOpen || (m.screen == screenLibrary && m.w >= 70)
	if m.eqOpen {
		showQueue = false
	}
	if showQueue {
		l.leftW = min(38, max(28, m.w*28/100))
		if m.queueOpen {
			l.leftW = m.w
		}
		l.rightW = m.w - l.leftW
		l.qw = l.leftW - 2
		if m.screen == screenLibrary && !m.queueOpen && len(m.recent) > 0 {
			l.recH = recentPanelHeight(bodyH)
		}
		l.qh = bodyH - 2 - l.recH
		l.queueBody = rect{1, l.queueTop(), l.qw, l.queueRows()}
		l.recentGrid = recentGrid(l.qw, l.recH-3)
		if l.recH > 0 && l.recentGrid.artCols > 0 {
			g := l.recentGrid
			l.recentBody = rect{1, l.recentTop(), l.qw, l.recH - 3}
			for c := range l.recentTiles {
				l.recentTiles[c] = rect{1 + g.gap*(c+1) + g.artCols*c, l.recentTop(), g.artCols, g.artRows + 2}
			}
		} else if l.recH > 0 {
			l.recH = 0
			l.qh = bodyH - 2
			l.queueBody.h = l.queueRows()
		}
		l.main = rect{l.leftW, 2, l.rightW, bodyH}
	}
	if !m.queueOpen && !m.eqOpen && m.screen == screenLibrary {
		l.libraryBody = rect{l.main.x + 1, l.main.y + 3, l.main.w - 2, l.main.h - 4}
		x = l.main.x + 1
		for i, name := range libraryNames {
			l.libraryTabs[i] = rect{x, l.main.y + 2, lipgloss.Width(name) + 2, 1}
			x += lipgloss.Width(name) + 3
		}
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
// screen row the first of them is drawn on. Sections reserve one breathing
// row and one heading row before the body starts.
func (l layout) queueRows() int { return max(0, l.qh-1) }
func (l layout) queueTop() int  { return l.queueY + 2 }

// recentTop is the screen row the cover grid starts on, directly below the
// queue box.
func (l layout) recentTop() int { return l.queueY + l.qh + 2 + 2 }

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

// recentPanelHeight is the on-screen height of the recently-played grid for a
// given top-area height, or 0 when the terminal is too short to fit both the
// grid and a usable queue.
func recentPanelHeight(topH int) int {
	h := topH * 46 / 100
	if h > 16 {
		h = 16 // past this the covers stop growing — width is the binding limit
	}
	if h < 9 {
		return 0 // tiles this short are unreadable — give the queue the space
	}
	return h
}

// recentVisible reports whether the grid is on screen and therefore focusable.
func (m model) recentVisible() bool {
	return m.layout().recH > 0
}
