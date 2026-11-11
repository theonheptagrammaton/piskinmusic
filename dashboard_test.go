package main

import (
	"image/color"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/k1y0miiii/applemusic-tui/engine"
	"github.com/k1y0miiii/applemusic-tui/lyrics"
	"github.com/k1y0miiii/applemusic-tui/visualizer"
	"github.com/muesli/termenv"
)

func dashboardFixture() model {
	m := recentModel()
	m.themeName = "piskin"
	m.art = solidImage(32, 32, color.RGBA{210, 60, 30, 255})
	m.st.Now.Title, m.st.Now.Artist = "TRACK", "ARTIST"
	m.st.Queue = []engine.Track{{Title: "SONG", Duration: time.Minute}}
	m.library.Albums = []engine.Track{{Title: "RECORD"}}
	m.ly = lyrics.Lyrics{Synced: true, Lines: []lyrics.Line{{At: 0, Text: "VERSE"}}}
	return m
}

// Check actual content inside every panel, not just the presence of its label.
// In particular, 40×12 must retain one cover row, one lyric and usable lists.
func TestDashboardKeepsFiveRegionsAndContentThroughResize(t *testing.T) {
	defer lipgloss.SetColorProfile(lipgloss.ColorProfile())
	lipgloss.SetColorProfile(termenv.TrueColor)
	m := dashboardFixture()
	for _, size := range [][2]int{{40, 12}, {41, 13}, {60, 20}, {80, 24}, {99, 12}, {100, 12}, {120, 32}, {180, 50}, {40, 12}} {
		m.w, m.h = size[0], size[1]
		for s := screenLibrary; s <= screenLyrics; s++ {
			m.selectScreen(s)
			l := m.layout()
			rows := strings.Split(m.View(), "\n")
			for r, want := range map[rect]string{
				l.library: "RECORD", l.queue: "SONG", l.nowPlaying: "▀",
				l.visualizer: "unavailable", l.lyrics: "VERSE",
			} {
				var content strings.Builder
				for y := r.y + 1; y < r.y+r.h; y++ {
					content.WriteString(ansi.Strip(ansi.Cut(rows[y], r.x, r.x+r.w)))
				}
				if !strings.Contains(content.String(), want) {
					t.Fatalf("%v highlight %d: panel %+v lost %q: %q", size, s, r, want, content.String())
				}
			}
		}
	}
}

func TestRainbowReachesAllVisualizersAndSelectedRows(t *testing.T) {
	defer lipgloss.SetColorProfile(lipgloss.ColorProfile())
	defer applyTheme(themes[0])
	lipgloss.SetColorProfile(termenv.TrueColor)
	applyTheme(themes[0])
	m := dashboardFixture()
	m.vizLive, m.vizSource = true, "FIXTURE"
	for i := range m.vizBands {
		m.vizBands[i] = 0.8
	}
	m.vizPCM = visualizer.Frame{Format: visualizer.Format{SampleRate: 48000, Channels: 2},
		Waveform: visualizer.Waveform{Count: 1, Frames: 4096}}
	m.vizPCM.Waveform.Points[0] = visualizer.WaveformPoint{Min: [2]float32{-0.8, -0.8}, Max: [2]float32{0.8, 0.8}}
	for mode := 0; mode < vizModes; mode++ {
		m.vizMode = mode
		frame := m.vizPanel(60, 14)
		hues := map[string]bool{}
		for _, c := range regexp.MustCompile(`38;2;(\d+);(\d+);(\d+)`).FindAllStringSubmatch(frame, -1) {
			if c[1] != c[2] || c[2] != c[3] {
				hues[c[0]] = true
			}
		}
		if len(hues) < 3 {
			t.Fatalf("visualizer %d lost its spectrum (%d hues)", mode, len(hues))
		}
	}
	row := selectedRow("▸ Selected recording", 30)
	if !strings.Contains(row, "38;2;255;112;125") || !strings.Contains(row, "38;2;238;138;255") {
		t.Fatal("selection lost its rainbow")
	}
}

func TestCompactDemoLabelRemainsExplicit(t *testing.T) {
	defer lipgloss.SetColorProfile(lipgloss.ColorProfile())
	lipgloss.SetColorProfile(termenv.Ascii)
	m := dashboardFixture()
	m.w, m.h, m.screen = 40, 12, screenVisualizer
	m.cfg = map[string]string{"visualizer.demo": "true"}
	if !strings.Contains(m.View(), "DEMO · SIMULATED") {
		t.Fatal("compact layout clipped the demo identity")
	}
}
