package main

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"runtime"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/k1y0miiii/applemusic-tui/engine"
	"github.com/muesli/termenv"
)

// Keep this harness identical when measuring the parent and optimized source.
// It excludes browser/capture cost, network and terminal I/O; it measures the
// actual Update/View path with deterministic, original-color artwork.
func performanceModel(s screen, playing, reduced bool) model {
	img := image.NewRGBA(image.Rect(0, 0, 256, 256))
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			img.SetRGBA(x, y, color.RGBA{uint8(x), uint8(y), 127, 255})
		}
	}
	m := model{phase: phaseReady, screen: s, art: img, artFor: "fixture",
		cfg:     map[string]string{"ui.reduced_motion": fmt.Sprint(reduced)},
		vizLive: true, vizSource: "FIXTURE", cmds: commandSet{tick: func() tea.Cmd { return nil }},
		st: engine.State{Playing: playing, Authed: true, Dur: 4 * time.Minute,
			Now: engine.Track{ID: "fixture", Title: "Performance fixture", Artist: "Synthetic metadata"}},
	}
	for i := range m.vizTargets {
		m.vizTargets[i] = float64(i+1) / 32
	}
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	return next.(model)
}

func BenchmarkTarget(b *testing.B) {
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(previous)
	for _, tc := range []struct {
		name                   string
		s                      screen
		playing, reduced, help bool
	}{
		{"FocusPlaying", screenFocus, true, false, false},
		{"LyricsPlaying", screenLyrics, true, false, false},
		{"VisualizerPlaying", screenVisualizer, true, false, false},
		{"LibraryPlaying", screenLibrary, true, false, false},
		{"FocusIdle", screenFocus, false, false, false},
		{"VisualizerReduced", screenVisualizer, true, true, false},
		{"CoveredVisualizer", screenVisualizer, true, false, true},
	} {
		b.Run(tc.name, func(b *testing.B) {
			m := performanceModel(tc.s, tc.playing, tc.reduced)
			m.helpOpen = tc.help
			_ = m.View() // warm static caches, matching steady-state playback
			now := time.Unix(1700000000, 0)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				now = now.Add(time.Second / 30)
				next, _ := m.Update(tickMsg(now))
				m = next.(model)
				_ = m.View()
			}
		})
	}
}

func BenchmarkTargetLayout(b *testing.B) {
	m := performanceModel(screenLibrary, true, false)
	_ = m.layout()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.layout()
	}
}

// Opt-in real timer loop, using the same harness on both revisions. This is a
// synthetic UI workload on target hardware, not a playback/capture benchmark.
func TestTargetCadenceMeasurement(t *testing.T) {
	if os.Getenv("AMTUI_MEASURE_UI") != "1" {
		t.Skip("set AMTUI_MEASURE_UI=1 for a timed target UI measurement")
	}
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(previous)
	for _, tc := range []struct {
		name                      string
		s                         screen
		playing, reduced, blurred bool
	}{
		{"FocusPlaying", screenFocus, true, false, false},
		{"FocusIdle", screenFocus, false, false, false},
		{"VisualizerPlaying", screenVisualizer, true, false, false},
		{"VisualizerReduced", screenVisualizer, true, true, false},
		{"BlurredVisualizer", screenVisualizer, true, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := performanceModel(tc.s, tc.playing, tc.reduced)
			m.cmds.tick = nil // use the production timer policy
			_ = m.View()
			if tc.blurred {
				next, _ := m.Update(tea.BlurMsg{})
				m = next.(model)
			}
			runtime.GC()
			var before, after runtime.MemStats
			runtime.ReadMemStats(&before)
			start := time.Now()
			frames := 0
			for time.Since(start) < 3*time.Second {
				msg := m.commands().tick()()
				next, _ := m.Update(msg)
				m = next.(model)
				_ = m.View()
				frames++
			}
			elapsed := time.Since(start)
			runtime.ReadMemStats(&after)
			t.Logf("frames=%d elapsed=%s Hz=%.2f allocated_bytes=%d allocations=%d retained_heap=%d", frames, elapsed, float64(frames)/elapsed.Seconds(), after.TotalAlloc-before.TotalAlloc, after.Mallocs-before.Mallocs, after.HeapAlloc)
		})
	}
}
