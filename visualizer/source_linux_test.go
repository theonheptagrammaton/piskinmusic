//go:build linux

package visualizer

import (
	"errors"
	"testing"

	"github.com/jfreymuth/pulse/proto"
)

// Synthetic identity cases; these do not exercise a running audio server.
func TestPulseSourceName(t *testing.T) {
	for _, tt := range []struct{ server, want string }{
		{"PulseAudio (on PipeWire 1.6.9)", "PIPEWIRE"},
		{"PipeWire", "PIPEWIRE"},
		{"PipeWire 1.6.9", "PIPEWIRE"},
		{"pulseaudio", "PULSEAUDIO"},
		{"PulseAudio", "PULSEAUDIO"},
		{"", "PULSEAUDIO"},
		{"some server mentioning PipeWire", "PULSEAUDIO"},
	} {
		t.Run(tt.server, func(t *testing.T) {
			if got := pulseSourceName(tt.server); got != tt.want {
				t.Fatalf("pulseSourceName(%q) = %q, want %q", tt.server, got, tt.want)
			}
		})
	}
}

type pulseRequestFunc func(proto.RequestArgs, proto.Reply) error

func (f pulseRequestFunc) RawRequest(req proto.RequestArgs, reply proto.Reply) error {
	return f(req, reply)
}

// Synthetic protocol boundary checks, including errors Start used to ignore.
func TestStartPulseRecord(t *testing.T) {
	for _, failAt := range []int{0, 1, 2} {
		calls := 0
		failure := errors.New("record request rejected")
		request := pulseRequestFunc(func(req proto.RequestArgs, _ proto.Reply) error {
			calls++
			switch req := req.(type) {
			case *proto.FlushRecordStream:
				if calls != 1 || req.StreamIndex != 42 {
					t.Fatalf("unexpected flush: call=%d index=%d", calls, req.StreamIndex)
				}
			case *proto.CorkRecordStream:
				if calls != 2 || req.StreamIndex != 42 || req.Corked {
					t.Fatalf("unexpected uncork: call=%d index=%d corked=%t", calls, req.StreamIndex, req.Corked)
				}
			default:
				t.Fatalf("unexpected request %T", req)
			}
			if calls == failAt {
				return failure
			}
			return nil
		})
		err := startPulseRecord(request, 42)
		if failAt == 0 {
			if err != nil || calls != 2 {
				t.Fatalf("start = %v, calls=%d", err, calls)
			}
		} else if !errors.Is(err, failure) || calls != failAt {
			t.Fatalf("failure at %d: err=%v calls=%d", failAt, err, calls)
		}
	}
}
