//go:build linux

package main

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/k1y0miiii/applemusic-tui/visualizer"
)

// Actual system capture through Service -> model -> rendered terminal frame.
// This can verify PCM during controlled playback, not audibility or entitlement.
func TestPCMLiveUI(t *testing.T) {
	if os.Getenv("AMTUI_TEST_PULSE") != "1" {
		t.Skip("set AMTUI_TEST_PULSE=1 for live Linux capture and waveform UI")
	}
	source, err := visualizer.OpenSystemSource()
	if err != nil {
		t.Fatal(err)
	}
	service, err := visualizer.NewService(source)
	if err != nil {
		_ = source.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = service.Close() })
	frame := waitForVisualizerFrame(t, service)
	if service.Err() != nil || time.Since(frame.At) > visualizerFrameFreshness {
		t.Fatal("capture was not fresh")
	}
	next, _ := (model{phase: phaseReady, vizOpening: true, screen: screenVisualizer}).Update(
		visualizerOpenedMsg{service: service, source: source.Name()})
	next, _ = next.(model).Update(tickMsg(time.Now()))
	m := next.(model)
	if m.captureStatus() != captureLive || m.vizPCM.Waveform.Count == 0 || m.vizSource != source.Name() {
		t.Fatal("live capture did not reach the UI with PCM metadata")
	}
	if os.Getenv("AMTUI_TEST_PULSE_REQUIRE_SIGNAL") == "1" && m.vizPCM.Waveform.Peak == [2]float64{} {
		t.Fatal("expected actual PCM signal during controlled playback")
	}
	for _, size := range [][2]int{{40, 12}, {80, 24}, {120, 32}} {
		m.w, m.h = size[0], size[1]
		for mode := 0; mode < vizModes; mode++ {
			m.vizMode = mode
			view := m.View()
			if lipgloss.Width(view) != m.w || lipgloss.Height(view) != m.h ||
				!strings.Contains(view, "LIVE") || !strings.Contains(view, source.Name()) {
				t.Fatalf("mode%d size%v lost live source label or frame bounds", mode, size)
			}
			if mode == vizWaveform && !strings.Contains(view, "PCM") {
				t.Fatalf("size%v did not render PCM waveform", size)
			}
		}
	}
	t.Logf("live UI source=%s format=%v total_frames=%d points=%d window_frames=%d peak=%v rms=%v",
		m.vizSource, m.vizPCM.Format, m.vizPCM.TotalFrames, m.vizPCM.Waveform.Count,
		m.vizPCM.Waveform.Frames, m.vizPCM.Waveform.Peak, m.vizPCM.Waveform.RMS)
}
