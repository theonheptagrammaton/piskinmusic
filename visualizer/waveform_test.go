package visualizer

import (
	"context"
	"math"
	"testing"
	"time"
)

// These exercise Source -> Service -> Frame using controlled PCM, not hardware.
func TestPCMSilenceAndSine(t *testing.T) {
	for _, channels := range []int{1, 2} {
		format := Format{SampleRate: 48_000, Channels: channels}
		silence := pcmFrame(t, format, make([]float32, windowFrames*channels), []int{4096})
		if silence.Waveform != (Waveform{Count: WaveformPoints, Frames: windowFrames}) {
			t.Fatalf("silence must have zero levels and envelope: %+v", silence.Waveform)
		}
		if !silence.Live || silence.At.IsZero() || silence.Format != format || silence.TotalFrames != windowFrames {
			t.Fatalf("silence is valid live PCM: %+v", silence)
		}
		// 1024Hz at 32768Hz has exactly 128 whole cycles in the window.
		format.SampleRate = 32768
		frame := pcmFrame(t, format, sinePCM(32768, channels, 1024, 0.5, windowFrames, false), []int{4096})
		for ch := 0; ch < channels; ch++ {
			if math.Abs(frame.Waveform.RMS[ch]-0.5/math.Sqrt2) > 1e-7 || frame.Waveform.Peak[ch] != 0.5 {
				t.Fatalf("sine ch%d RMS=%g peak=%g", ch, frame.Waveform.RMS[ch], frame.Waveform.Peak[ch])
			}
			for i, point := range frame.Waveform.Points {
				if point.Min[ch] != -0.5 || point.Max[ch] != 0.5 {
					t.Fatalf("sine point%d ch%d = %v", i, ch, point)
				}
			}
		}
	}
}

func TestPCMImpulseSurvivesAggregation(t *testing.T) {
	for _, offset := range []int{0, 31, 32, 2047, 4095} {
		for _, amplitude := range []float32{-1, 1} {
			pcm := make([]float32, windowFrames)
			pcm[offset] = amplitude
			frame := pcmFrame(t, Format{48000, 1}, pcm, []int{7, 23, 101, 3})
			for i, point := range frame.Waveform.Points {
				want := WaveformPoint{}
				if i == offset/32 {
					want.Min[0], want.Max[0] = min(0, amplitude), max(0, amplitude)
				}
				if point != want {
					t.Fatalf("impulse at %d point %d=%v want %v", offset, i, point, want)
				}
			}
			if frame.Waveform.RMS[0] != 1.0/64 || frame.Waveform.Peak[0] != 1 {
				t.Fatalf("impulse levels=%v %v", frame.Waveform.RMS, frame.Waveform.Peak)
			}
		}
	}
}

func TestPCMStereoAndAntiPhaseStaySeparate(t *testing.T) {
	for _, right := range []float32{0.25, -0.75} {
		pcm := make([]float32, 8192)
		for i := 0; i < len(pcm); i += 2 {
			pcm[i], pcm[i+1] = 0.75, right
		}
		frame := pcmFrame(t, Format{48000, 2}, pcm, []int{13, 411, 1})
		for _, point := range frame.Waveform.Points {
			if point.Min != [2]float32{0.75, right} || point.Max != point.Min {
				t.Fatalf("stereo channel amplitudes lost: %v", point)
			}
		}
		if frame.Waveform.RMS != [2]float64{0.75, math.Abs(float64(right))} {
			t.Fatalf("stereo levels lost: %v", frame.Waveform.RMS)
		}
	}
}

func TestPCMChunkingAndRollingWindow(t *testing.T) {
	// A ramp makes chronology and exact min/max bounds visible after wrapping.
	const frames = 10003
	pcm := make([]float32, frames*2)
	for i := range frames {
		pcm[2*i] = float32(i) / frames
		pcm[2*i+1] = -pcm[2*i]
	}
	whole := pcmFrame(t, Format{48000, 2}, pcm, []int{4096})
	chunks := pcmFrame(t, Format{48000, 2}, pcm, []int{1, 17, 93, 501, 11})
	if whole.Waveform != chunks.Waveform || whole.Bands != chunks.Bands || chunks.TotalFrames != frames {
		t.Fatal("read chunking changed the envelope, metadata or existing spectrum")
	}
	for i, point := range chunks.Waveform.Points {
		lo, hi := frames-windowFrames+i*32, frames-windowFrames+(i+1)*32-1
		if point.Min != [2]float32{pcm[2*lo], pcm[2*hi+1]} ||
			point.Max != [2]float32{pcm[2*hi], pcm[2*lo+1]} {
			t.Fatalf("rolling point %d out of order: %v", i, point)
		}
	}
	short := pcmFrame(t, Format{44100, 1}, []float32{-0.5, 0.25, 1}, []int{1})
	if short.Waveform.Count != 3 || short.Waveform.Frames != 3 || short.TotalFrames != 3 ||
		short.Waveform.Points[0].Min[0] != -0.5 || short.Waveform.Points[2].Max[0] != 1 {
		t.Fatal("partial startup window padded with invented PCM")
	}
}

func TestPCMFrameOwnsValuesAcrossBufferReuseAndLatest(t *testing.T) {
	advance := make(chan struct{})
	reads := 0
	source := &fakeSource{name: "fixture", format: Format{48000, 1}}
	source.readFn = func(ctx context.Context, dst []float32) (int, error) {
		reads++
		if reads == 2 {
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			case <-advance:
			}
		} else if reads > 2 {
			<-ctx.Done()
			return 0, ctx.Err()
		}
		for i := range dst {
			dst[i] = float32(reads) / 4
		}
		return len(dst), nil
	}
	service, err := NewService(source)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = service.Close() })
	first := waitForFrame(t, service)
	mutated := first
	mutated.Waveform.Points[0].Min[0] = -99
	got, _ := service.Latest()
	if got != first {
		t.Fatal("Latest exposed mutable storage")
	}
	close(advance)
	deadline := time.Now().Add(time.Second)
	for {
		got, _ = service.Latest()
		if got.TotalFrames == 8192 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("no second PCM frame")
		}
		time.Sleep(time.Millisecond)
	}
	if first.Waveform.Points[0].Min[0] != 0.25 || got.Waveform.Points[0].Min[0] != 0.5 {
		t.Fatal("capture buffer reuse corrupted an owned frame")
	}
}

// chunks contains sample counts, including reads that split stereo frames.
func pcmFrame(t *testing.T, format Format, pcm []float32, chunks []int) Frame {
	t.Helper()
	offset, read := 0, 0
	source := &fakeSource{name: "deterministic PCM", format: format}
	source.readFn = func(ctx context.Context, dst []float32) (int, error) {
		if offset == len(pcm) {
			<-ctx.Done()
			return 0, ctx.Err()
		}
		n := min(len(dst), len(pcm)-offset, chunks[read%len(chunks)])
		copy(dst, pcm[offset:offset+n])
		offset += n
		read++
		return n, nil
	}
	service, err := NewService(source)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = service.Close() })
	deadline := time.Now().Add(2 * time.Second)
	for {
		if err := service.Err(); err != nil {
			t.Fatal(err)
		}
		if frame, ok := service.Latest(); ok && frame.TotalFrames == uint64(len(pcm)/format.Channels) {
			if err := service.Close(); err != nil {
				t.Fatal(err)
			}
			return frame
		}
		if time.Now().After(deadline) {
			t.Fatal("no complete PCM snapshot")
		}
		time.Sleep(time.Millisecond)
	}
}
