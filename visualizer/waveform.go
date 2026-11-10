package visualizer

import "math"

const WaveformPoints = 128

// Waveform is a value-owned envelope of the most recent 4096 PCM frames
// (about 85ms at 48kHz). Only Points[:Count] and channels in Frame.Format
// are valid. Each point covers [i*Frames/Count, (i+1)*Frames/Count) in time.
// Channels stay separate so anti-phase stereo cannot cancel. Values retain
// the source's signed amplitude; renderers may clip to full scale [-1, 1].
type Waveform struct {
	Points [WaveformPoints]WaveformPoint
	Count  int
	Frames int
	RMS    [2]float64 // per-channel levels over the same window
	Peak   [2]float64
}

type WaveformPoint struct {
	Min [2]float32
	Max [2]float32
}

// pcmWindow is owned exclusively by Service.run. Analyzer validates samples
// before they enter this ring; neither capture slices nor this ring escape.
type pcmWindow struct {
	ring  [windowFrames][2]float32
	write int
	count int
	total uint64
}

func (p *pcmWindow) append(samples []float32, channels int) {
	for base := 0; base < len(samples); base += channels {
		for ch := 0; ch < channels; ch++ {
			p.ring[p.write][ch] = samples[base+ch]
		}
		p.write = (p.write + 1) % windowFrames
		p.count = min(windowFrames, p.count+1)
		p.total++
	}
}

func (p *pcmWindow) snapshot(channels int) Waveform {
	w := Waveform{Frames: p.count, Count: min(WaveformPoints, p.count)}
	start := (p.write - p.count + windowFrames) % windowFrames
	for point := 0; point < w.Count; point++ {
		lo, hi := point*p.count/w.Count, (point+1)*p.count/w.Count
		for i := lo; i < hi; i++ {
			values := p.ring[(start+i)%windowFrames]
			for ch := 0; ch < channels; ch++ {
				v := values[ch]
				if i == lo {
					w.Points[point].Min[ch], w.Points[point].Max[ch] = v, v
				} else {
					w.Points[point].Min[ch] = min(w.Points[point].Min[ch], v)
					w.Points[point].Max[ch] = max(w.Points[point].Max[ch], v)
				}
				w.RMS[ch] += float64(v) * float64(v)
				w.Peak[ch] = max(w.Peak[ch], math.Abs(float64(v)))
			}
		}
	}
	if p.count > 0 {
		for ch := 0; ch < channels; ch++ {
			w.RMS[ch] = math.Sqrt(w.RMS[ch] / float64(p.count))
		}
	}
	return w
}
