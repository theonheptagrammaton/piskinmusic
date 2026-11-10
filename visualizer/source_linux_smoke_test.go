//go:build linux

package visualizer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"testing"
	"time"
)

// Live hardware only: no generated samples or fallback source. The default
// accepts silence; REQUIRE_SIGNAL makes an explicitly controlled playback run
// fail on silence. Neither mode proves subscription playback by itself.
func TestPulseLiveSmoke(t *testing.T) {
	if os.Getenv("AMTUI_TEST_PULSE") != "1" {
		t.Skip("set AMTUI_TEST_PULSE=1 to exercise the live Linux monitor")
	}
	for _, useService := range []bool{false, true} {
		name := "Source"
		if useService {
			name = "Service"
		}
		t.Run(name, func(t *testing.T) {
			source, err := OpenSystemSource()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := source.Close(); err != nil {
					t.Errorf("source close: %v", err)
				}
			})
			format := source.Format()
			if err := validateSourceFormat(format); err != nil {
				t.Fatal(err)
			}
			t.Logf("source=%s negotiated=float32 %dHz %dch start=%s",
				source.Name(), format.SampleRate, format.Channels, time.Now().Format(time.RFC3339Nano))
			meter := &pulseSmokeMeter{Source: source}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if useService {
				service, err := NewService(meter)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					if err := service.Close(); err != nil {
						t.Errorf("service close: %v", err)
					}
				})
				ticker := time.NewTicker(20 * time.Millisecond)
				defer ticker.Stop()
				var first, last time.Time
				frames, changes := 0, 0
				var previous [32]float64
				var peakBand float64
			loop:
				for {
					select {
					case <-ctx.Done():
						break loop
					case <-ticker.C:
					}
					if err := service.Err(); err != nil {
						t.Fatalf("service: %v", err)
					}
					frame, ok := service.Latest()
					if !ok || frame.At.Equal(last) {
						continue
					}
					age := time.Since(frame.At)
					if !frame.Live || frame.Source != source.Name() || frame.At.IsZero() ||
						age < 0 || age > 400*time.Millisecond || (!last.IsZero() && !frame.At.After(last)) {
						t.Fatalf("invalid frame metadata: source=%q live=%t at=%s age=%s", frame.Source, frame.Live, frame.At, age)
					}
					for _, band := range frame.Bands {
						if math.IsNaN(band) || math.IsInf(band, 0) || band < 0 || band > 1 {
							t.Fatalf("invalid FFT band: %g", band)
						}
						peakBand = max(peakBand, band)
					}
					if first.IsZero() {
						first = frame.At
					} else if frame.Bands != previous {
						changes++
					}
					last, previous = frame.At, frame.Bands
					frames++
				}
				// Close joins the reader before inspecting its aggregate counters.
				if err := service.Close(); err != nil {
					t.Fatal(err)
				}
				if err := service.Err(); err != nil {
					t.Fatal(err)
				}
				t.Logf("frames=%d changing_spectra=%d first=%s last=%s peak_band=%.6g", frames, changes,
					first.Format(time.RFC3339Nano), last.Format(time.RFC3339Nano), peakBand)
				if frames < 2 || time.Since(last) > 400*time.Millisecond {
					t.Error("no sustained fresh service frames")
				}
				if os.Getenv("AMTUI_TEST_PULSE_REQUIRE_SIGNAL") == "1" && (peakBand == 0 || changes == 0) {
					t.Error("expected changing FFT signal during controlled playback")
				}
			} else {
				buffer := make([]float32, 8192)
				for {
					_, err := meter.Read(ctx, buffer)
					if errors.Is(err, context.DeadlineExceeded) {
						break
					}
					if err != nil {
						t.Fatal(err)
					}
				}
				cancelled, cancelRead := context.WithCancel(context.Background())
				cancelRead()
				if n, err := source.Read(cancelled, buffer); n != 0 || !errors.Is(err, context.Canceled) {
					t.Fatalf("cancelled read = (%d, %v)", n, err)
				}
				if err := source.Close(); err != nil {
					t.Fatal(err)
				}
				if n, err := source.Read(context.Background(), buffer); n != 0 || !errors.Is(err, io.ErrClosedPipe) {
					t.Fatalf("read after close = (%d, %v)", n, err)
				}
			}
			meter.check(t)
		})
	}
}

// Only aggregate counts and levels survive Read. No PCM buffers are retained.
type pulseSmokeMeter struct {
	Source
	reads, samples, nonfinite int
	squares, peak             float64
	first, last               time.Time
}

func (m *pulseSmokeMeter) Read(ctx context.Context, dst []float32) (int, error) {
	n, err := m.Source.Read(ctx, dst)
	if n < 0 || n > len(dst) || n%m.Format().Channels != 0 {
		return 0, fmt.Errorf("invalid PCM sample count %d", n)
	}
	if n > 0 {
		m.reads++
		m.samples += n
		m.last = time.Now()
		if m.first.IsZero() {
			m.first = m.last
		}
		for _, sample := range dst[:n] {
			v := float64(sample)
			if math.IsNaN(v) || math.IsInf(v, 0) {
				m.nonfinite++
				continue
			}
			m.squares += v * v
			m.peak = max(m.peak, math.Abs(v))
		}
	}
	return n, err
}

func (m *pulseSmokeMeter) check(t *testing.T) {
	t.Helper()
	rms := math.Sqrt(m.squares / float64(max(1, m.samples)))
	t.Logf("reads=%d samples=%d nonfinite=%d rms=%.9g peak=%.9g first=%s last=%s",
		m.reads, m.samples, m.nonfinite, rms, m.peak,
		m.first.Format(time.RFC3339Nano), m.last.Format(time.RFC3339Nano))
	if m.reads < 2 || m.samples == 0 || m.nonfinite != 0 || time.Since(m.last) > 400*time.Millisecond {
		t.Fatal("expected sustained, fresh, nonempty, finite PCM reads")
	}
	if os.Getenv("AMTUI_TEST_PULSE_REQUIRE_SIGNAL") == "1" && rms <= 1e-6 {
		t.Fatal("expected nonzero PCM during controlled playback")
	}
}
