package main

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestResizeKeepsRenderedTilesAndHitAreasTogether(t *testing.T) {
	defer lipgloss.SetColorProfile(lipgloss.ColorProfile())
	lipgloss.SetColorProfile(termenv.Ascii)
	m := recentModel()
	p := &commandPlayback{}
	m.eng, m.focus = p, focusRecent
	for i := range m.recent {
		m.recent[i].Title = fmt.Sprintf("Tile%d", i)
	}
	m.recentSel = 2
	// Shrink below each threshold and grow back without resetting selection.
	for _, size := range [][2]int{{120, 35}, {70, 22}, {160, 48}, {69, 35}, {120, 21}, {0, 0}, {80, 28}, {120, 35}} {
		next, cmd := m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		m = next.(model)
		if cmd != nil {
			t.Fatal("resize scheduled effects")
		}
		l := m.layout()
		if !l.visible {
			if m.w > 0 && m.h > 0 && !strings.Contains(m.View(), "terminal too small") {
				t.Fatal("hidden layout rendered controls")
			}
			// Even a coordinate that would have hit the old queue is inert.
			next, cmd := m.Update(tea.MouseMsg{X: 5, Y: 2, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
			if cmd != nil || next.(model).selIdx != m.selIdx {
				t.Fatal("hidden queue received a click")
			}
			continue
		}
		rows := strings.Split(m.View(), "\n")
		if len(rows) != m.h {
			t.Fatalf("%v rendered %d rows", size, len(rows))
		}
		for _, row := range rows {
			if lipgloss.Width(row) > m.w {
				t.Fatalf("%v overflowed", size)
			}
		}
		for c, tile := range l.recentTiles {
			if tile.w == 0 {
				continue
			}
			want := m.recentSel/recentCols*recentCols + c
			captionY := tile.y + l.recentGrid.artRows
			if !strings.Contains(rows[captionY], fmt.Sprintf("Tile%d", want)) {
				t.Fatalf("%v: missing tile %d on row %d", size, want, captionY)
			}
			next, cmd := m.Update(tea.MouseMsg{X: tile.x, Y: captionY, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
			if cmd == nil || next.(model).recentSel != want {
				t.Fatalf("%v: click missed drawn tile %d", size, want)
			}
		}
		if !m.recentVisible() && m.focus == focusRecent {
			t.Fatal("resize left focus in hidden grid")
		}
	}
	if len(p.calls) != 0 {
		t.Fatalf("render/resize performed playback work: %v", p.calls)
	}
}

func TestMouseIgnoresBordersGapsAndBlankTileRows(t *testing.T) {
	m := recentModel()
	m.selIdx, m.recentSel = 2, 2
	l := m.layout()
	for _, point := range [][2]int{
		{-1, l.queueTop()}, {0, l.queueTop()}, {l.leftW - 1, l.queueTop()},
		{1, l.recentTop()}, // gap before the covers
		{l.recentTiles[0].x, l.recentBody.y + l.recentBody.h},         // bottom border
		{l.recentTiles[0].x, l.recentTiles[0].y + l.recentTiles[0].h}, // blank padding below tiles
	} {
		next, cmd := m.Update(tea.MouseMsg{X: point[0], Y: point[1], Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
		got := next.(model)
		if cmd != nil || got.selIdx != m.selIdx || got.recentSel != m.recentSel {
			t.Fatalf("off-content click %v activated playback", point)
		}
	}
}
