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
		{"demo", model{}, captureDemo},
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
