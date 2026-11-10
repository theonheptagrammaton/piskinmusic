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

type layout struct {
	visible                                    bool
	queueBody, recentBody, transport, progress rect

	recentGrid  recentGridLayout
	recentTiles [recentCols]rect

	topH   int // rows above the transport
	leftW  int // full width of the queue column, borders included
	rightW int
	qw, qh int // queue panel content size
	recH   int // full height of the recently-played box, 0 when hidden
	vw, vh int // visualizer panel
	lh     int // lyrics panel content height
}

func (m model) layout() layout {
	l := layout{visible: m.w >= 70 && m.h >= 22, topH: m.h - 4}
	if !l.visible {
		return l
	}
	l.leftW = m.w * 42 / 100
	l.rightW = m.w - l.leftW
	l.qw = l.leftW - 2
	if len(m.recent) > 0 {
		l.recH = recentPanelHeight(l.topH)
	}
	l.qh = l.topH - 2 - l.recH
	l.vh = l.topH * 62 / 100
	l.vw = l.rightW - 2
	l.lh = l.topH - l.vh - 2
	l.queueBody = rect{1, l.queueTop(), l.qw, l.queueRows()}
	l.recentGrid = recentGrid(l.qw, l.recH-3)
	if l.recH > 0 {
		l.recentBody = rect{1, l.recentTop(), l.qw, l.topH - 1 - l.recentTop()}
		g := l.recentGrid
		for c := range l.recentTiles {
			l.recentTiles[c] = rect{1 + g.gap*(c+1) + g.artCols*c, l.recentTop(), g.artCols, g.artRows + 2}
		}
	}
	l.transport = rect{1, l.topH + 1, m.w - 2, 2}
	barX, barW := m.progressGeometry(l.transport.w)
	l.progress = rect{l.transport.x + barX, l.transport.y + 1, barW, 1}
	return l
}

// queueRows is how many tracks the queue panel shows at once, and queueTop the
// screen row the first of them is drawn on. panel() spends one row on the
// border and one on its title before the body starts.
func (l layout) queueRows() int { return l.qh - 1 }
func (l layout) queueTop() int  { return 2 }

// recentTop is the screen row the cover grid starts on, directly below the
// queue box.
func (l layout) recentTop() int { return l.qh + 2 + 2 }

// progressGeometry returns the column the progress bar starts at within the
// transport's content, and its width. transportPanel lays the second line out
// as " ▄▄ " then the play icon then the elapsed time, which is six cells plus
// the clock before the bar begins.
func (m model) progressGeometry(w int) (x, width int) {
	times := fmtTime(m.st.Pos) + " "
	timee := " " + fmtTime(m.st.Dur)
	shuf, rep := "off", [3]string{"off", "one", "all"}[min(m.st.Repeat, 2)]
	if m.st.Shuffle {
		shuf = "on"
	}
	right := fmt.Sprintf("⇄ %s  ↻ %s  vol ", shuf, rep) +
		strings.Repeat("▮", 6) + fmt.Sprintf(" %d%% ", m.st.Volume)
	barW := max(10, w-lipgloss.Width(times)-lipgloss.Width(timee)-lipgloss.Width(right)-8)
	return 6 + lipgloss.Width(times), barW
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
