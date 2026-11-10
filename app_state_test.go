package main

import (
	"testing"

	"github.com/k1y0miiii/applemusic-tui/visualizer"
)

func TestCaptureStatusDistinguishesLifecycleAndFallback(t *testing.T) {
	for _, tc := range []struct {
		name string
		m    model
		want captureStatus
	}{
		{"default unavailable", model{}, captureUnavailable},
		{"demo opt-in", model{cfg: map[string]string{"visualizer.demo": "true"}}, captureDemo},
		{"demo after failure", model{vizTerminal: true, cfg: map[string]string{"visualizer.demo": "true"}}, captureDemo},
		{"invalid demo stays off", model{cfg: map[string]string{"visualizer.demo": "invalid"}}, captureUnavailable},
		{"false demo stays off", model{cfg: map[string]string{"visualizer.demo": "false"}}, captureUnavailable},
		{"demo cannot replace live", model{vizLive: true, cfg: map[string]string{"visualizer.demo": "true"}}, captureLive},
		{"demo cannot replace stale", model{vizService: &visualizer.Service{}, cfg: map[string]string{"visualizer.demo": "true"}}, captureStale},
		{"demo cannot replace starting", model{vizOpening: true, cfg: map[string]string{"visualizer.demo": "true"}}, captureStarting},
		{"starting", model{vizOpening: true}, captureStarting},
		{"live silence", model{vizLive: true, vizSource: "PIPEWIRE"}, captureLive},
		{"stale", model{vizService: &visualizer.Service{}, vizSource: "PIPEWIRE"}, captureStale},
		{"unavailable", model{vizTerminal: true}, captureUnavailable},
		{"retry starting", model{vizTerminal: true, vizOpening: true}, captureStarting},
		{"recovered live", model{vizTerminal: true, vizLive: true}, captureLive},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.m.captureStatus(); got != tc.want {
				t.Fatalf("status = %v, want %v", got, tc.want)
			}
		})
	}
}
