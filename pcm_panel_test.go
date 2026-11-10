package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/k1y0miiii/applemusic-tui/visualizer"
)

func TestPCMPanelUsesSignedSeparateChannelsAndMetadata(t *testing.T) {
	frame := visualizer.Frame{
		Format:   visualizer.Format{SampleRate: 48000, Channels: 2},
		Waveform: visualizer.Waveform{Count: 1, Frames: 4096},
	}
	frame.Waveform.Points[0] = visualizer.WaveformPoint{
		Min: [2]float32{1, -1}, Max: [2]float32{1, -1},
	}
	got := pcmPanel(frame, 80, 11)
	lines := strings.Split(got, "\n")
	if !strings.Contains(lines[0], "PCM · system output · 48000 Hz · 2 ch · 4096 frames · 85.3 ms") {
		t.Fatalf("missing sample provenance: %s", lines[0])
	}
	if !strings.Contains(lines[1], "•") || !strings.Contains(lines[10], "•") ||
		!strings.HasPrefix(lines[3], "L ") || !strings.HasPrefix(lines[8], "R ") {
		t.Fatalf("positive left and negative right were not kept separate:\n%s", got)
	}
	// Shrinking merges min/max points, retaining an impulse between endpoints.
	frame.Format.Channels = 1
	frame.Waveform.Count = 128
	frame.Waveform.Points = [visualizer.WaveformPoints]visualizer.WaveformPoint{}
	frame.Waveform.Points[65].Max[0] = 1
	if got := pcmPanel(frame, 3, 6); !strings.Contains(strings.Split(got, "\n")[1], "│") {
		t.Fatalf("narrow PCM trace lost impulse: %s", got)
	}
}

func TestPCMPanelBoundsAndNonLiveModes(t *testing.T) {
	for _, size := range [][2]int{{0, 0}, {1, 1}, {3, 2}, {40, 8}, {120, 24}} {
		for mode := 0; mode < vizModes; mode++ {
			for _, m := range []model{
				{vizOpening: true}, {vizService: &visualizer.Service{}, vizSource: "PIPEWIRE"},
				{vizTerminal: true}, {cfg: map[string]string{"visualizer.demo": "true"}},
			} {
				m.vizMode = mode
				got := m.vizPanel(size[0], size[1])
				if lipgloss.Width(got) > size[0] || (got != "" && lipgloss.Height(got) > size[1]) {
					t.Fatalf("mode%d escaped %v bounds: %q", mode, size, got)
				}
				if mode == vizWaveform && strings.ContainsAny(got, "│•") {
					t.Fatalf("non-live mode invented PCM: %q", got)
				}
			}
		}
	}
}

func TestPCMErrorThenReopenRecoversWithRealSilence(t *testing.T) {
	fail := make(chan struct{})
	read := false
	terminal := errors.New("capture lost")
	source := &uiFakeSource{name: "PIPEWIRE", format: visualizer.Format{SampleRate: 48000, Channels: 1}}
	source.read = func(ctx context.Context, dst []float32) (int, error) {
		if !read {
			read = true
			fillSine(dst, 440)
			return len(dst), nil
		}
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-fail:
			return 0, terminal
		}
	}
	service, err := visualizer.NewService(source)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = service.Close() })
	frame := waitForVisualizerFrame(t, service)
	next, _ := (model{phase: phaseReady, vizService: service, vizMode: vizWaveform}).Update(tickMsg(frame.At))
	live := next.(model)
	if live.vizPCM.Waveform.Peak[0] == 0 {
		t.Fatal("no initial PCM signal")
	}
	close(fail)
	waitForVisualizerError(t, service, terminal)
	next, _ = live.Update(tickMsg(frame.At.Add(time.Millisecond)))
	failed := next.(model)
	if failed.captureStatus() != captureUnavailable || failed.vizPCM != (visualizer.Frame{}) ||
		strings.ContainsAny(failed.vizPanel(80, 12), "│•") || failed.note != "" {
		t.Fatal("terminal capture retained PCM or generated demo")
	}
	sent := false
	silentSource := &uiFakeSource{name: "PULSEAUDIO", format: visualizer.Format{SampleRate: 44100, Channels: 2}}
	silentSource.read = func(ctx context.Context, dst []float32) (int, error) {
		if !sent {
			sent = true
			clear(dst)
			return len(dst), nil
		}
		<-ctx.Done()
		return 0, ctx.Err()
	}
	reopened, err := visualizer.NewService(silentSource)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	failed.vizOpening = true // the ready/reconnect path requests a new source
	next, _ = failed.Update(visualizerOpenedMsg{service: reopened, source: "PULSEAUDIO"})
	newFrame := waitForVisualizerFrame(t, reopened)
	next, _ = next.(model).Update(tickMsg(newFrame.At))
	recovered := next.(model)
	if recovered.captureStatus() != captureLive || recovered.vizTerminal || recovered.vizPCM != newFrame ||
		recovered.vizSource != "PULSEAUDIO" || recovered.vizPCM.Waveform.Peak != [2]float64{} ||
		recovered.vizPCM.Format.SampleRate != 44100 {
		t.Fatal("reopened real silence did not recover fresh PCM with new source metadata")
	}
}

func TestPCMModePersistenceAndFreshnessBoundary(t *testing.T) {
	t.Setenv("AMTUI_CONFIG_DIR", t.TempDir())
	saveVizMode(vizWaveform)
	if loadVizMode() != vizWaveform || vizModeName(vizWaveform) != "waveform" {
		t.Fatal("waveform mode did not persist")
	}
	now := time.Now()
	if visualizerFrameStale(now, now.Add(-visualizerFrameFreshness)) ||
		visualizerFrameStale(now, now.Add(time.Millisecond)) ||
		!visualizerFrameStale(now, now.Add(-visualizerFrameFreshness-time.Nanosecond)) ||
		!visualizerFrameStale(now, time.Time{}) || !visualizerFrameStale(now, now.Add(time.Second)) {
		t.Fatal("freshness boundary accepts invalid timestamps or rejects fresh PCM")
	}
}
