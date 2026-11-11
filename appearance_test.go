package main

import (
	"image/color"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/k1y0miiii/applemusic-tui/lyrics"
	"github.com/muesli/termenv"
)

func TestHeaderAndTransportTargetsMatchVisibleLabels(t *testing.T) {
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(previous)
	for _, width := range []int{40, 49, 50, 51, 52, 53, 69, 70, 80, 120, 180} {
		for s := screenLibrary; s <= screenLyrics; s++ {
			m := recentModel()
			m.w, m.screen = width, s
			l := m.layout()
			rows := strings.Split(m.View(), "\n")
			check := func(r rect, label string) {
				t.Helper()
				if got := ansi.Strip(ansi.Cut(rows[r.y], r.x, r.x+r.w)); got != label {
					t.Fatalf("width %d screen %d: target %q contains %q", width, s, label, got)
				}
			}
			for i, r := range l.nav {
				check(r, m.screenLabel(i))
			}
			for _, action := range l.actions {
				check(action.rect, action.label)
				if action.x <= 13 {
					t.Fatal("toolbar overlaps the brand")
				}
			}
			check(l.play, "⏸")
			for _, action := range m.transportActions(l) {
				if action.label != "" {
					check(action.rect, action.label)
				}
			}
		}
	}
}

func TestPanelHighlightsKeepAllDashboardRegionsVisible(t *testing.T) {
	for _, width := range []int{70, 80, 120, 180, 240} {
		m := recentModel()
		m.w = width
		l := m.layout()
		if l.leftW > 38 || l.leftW >= width/2 || l.queueBody.w == 0 {
			t.Fatalf("width %d: sidebar crowds out the library: %+v", width, l)
		}
		for s := screenFocus; s <= screenLyrics; s++ {
			m.screen = s
			l = m.layout()
			if l.queueBody.h == 0 || l.libraryBody.h == 0 || l.nowPlaying.h == 0 || l.visualizer.h == 0 || l.lyrics.h == 0 {
				t.Fatalf("panel highlight %d hid dashboard content", s)
			}
			m.toggleQueue()
			if m.layout().queueBody.w != width-2 {
				t.Fatal("explicit queue no longer opens")
			}
			m.toggleQueue()
		}
	}
}

func TestFocusCentersArtworkAndPreservesMetadata(t *testing.T) {
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(previous)
	m := recentModel()
	m.art = solidImage(32, 32, color.RGBA{210, 60, 30, 255})
	m.st.Now.Album = "Random Access Memories"
	const w, h = 118, 28
	rows := strings.Split(ansi.Strip(m.focusPanel(w, h)), "\n")
	first, last, artX, titleY := -1, -1, -1, -1
	for y, row := range rows {
		if lipgloss.Width(row) != w {
			t.Fatal("focus composition overflows before frame clipping")
		}
		if x := strings.Index(row, "▀"); x >= 0 {
			if first < 0 {
				first, artX = y, x
			}
			last = y
		}
		if strings.Contains(row, m.st.Now.Title) {
			titleY = y
			if strings.Index(row, m.st.Now.Title) <= artX {
				t.Fatal("metadata is not beside the artwork")
			}
		}
	}
	if first < 1 || last >= len(rows)-1 || titleY <= first || titleY >= last {
		t.Fatalf("artwork and metadata are not vertically centered: first=%d last=%d title=%d", first, last, titleY)
	}
	if delta := first - (len(rows) - 1 - last); delta < -1 || delta > 1 {
		t.Fatal("artwork has unbalanced vertical margins")
	}
	if !strings.Contains(strings.Join(rows, "\n"), m.st.Now.Album) {
		t.Fatal("album metadata was lost")
	}
	for _, size := range [][2]int{{38, 4}, {58, 12}, {78, 18}, {118, 28}} {
		m.st.Now.Title = strings.Repeat("音楽 é 👩🏽‍🎤 ", 20)
		for _, enabled := range []string{"true", "false"} {
			m.cfg = map[string]string{"artwork.enabled": enabled}
			body := strings.Split(m.focusPanel(size[0], size[1]), "\n")
			if len(body) > size[1]-1 {
				t.Fatal("long focus metadata overflows vertically")
			}
			for _, row := range body {
				if lipgloss.Width(row) != size[0] {
					t.Fatal("Unicode metadata overflows horizontally")
				}
			}
		}
	}
}

func TestLongQueueTitlesKeepDurationsAndTransportHelpVisible(t *testing.T) {
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(previous)
	for _, width := range []int{26, 36, 68, 118} {
		for _, selected := range []bool{true, false} {
			row := listTrack("▸ ", strings.Repeat("音楽 é ", 30), "Artist", "5:37", width, selected, true)
			if lipgloss.Width(row) != width || !strings.HasSuffix(ansi.Strip(row), "5:37") {
				t.Fatalf("duration clipped by long Unicode title: %q", row)
			}
		}
	}
	m := recentModel()
	m.st.Now.Title = strings.Repeat("Long title ", 30)
	m.st.Pos, m.st.Dur = 83*time.Minute, 125*time.Minute
	for _, width := range []int{68, 78, 118} {
		body := ansi.Strip(m.transportPanel(width))
		if !strings.Contains(body, strings.TrimSpace(shortHints)) || !strings.Contains(body, "125:00") {
			t.Fatal("long metadata hid help or duration")
		}
	}
}

func TestSparseLyricsAreCenteredAndCurrentLineHasTextMarker(t *testing.T) {
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.Ascii)
	defer lipgloss.SetColorProfile(previous)
	m := recentModel()
	m.ly = lyrics.Lyrics{Synced: true, Lines: []lyrics.Line{
		{At: 0, Text: "First line"},
		{At: 30 * time.Second, Text: "Second line"},
		{At: 60 * time.Second, Text: "Current line"},
		{At: 120 * time.Second, Text: "Next line"},
	}}
	rows := strings.Split(m.lyricsText(80, 26), "\n")
	first, last, prior := -1, -1, -1
	for y, row := range rows {
		if strings.TrimSpace(row) == "" {
			continue
		}
		if first == -1 {
			first = y
		}
		if prior >= 0 && y-prior != 2 {
			t.Fatal("lyrics should have a breathing row between lines")
		}
		prior, last = y, y
	}
	if first < 1 || first-(len(rows)-1-last) < -1 || first-(len(rows)-1-last) > 1 {
		t.Fatal("short lyrics block is not vertically centered")
	}
	if !strings.Contains(strings.Join(rows, "\n"), "› Current line") {
		t.Fatal("current lyric needs a visible marker without color")
	}
}
