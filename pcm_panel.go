package main

import (
	"fmt"
	"math"
	"strings"

	"github.com/k1y0miiii/applemusic-tui/visualizer"
)

// pcmPanel renders only the immutable time-domain envelope. Spectrum bands,
// playback time and the track's loudness timeline do not enter this path.
func pcmPanel(frame visualizer.Frame, w, h int) string {
	pcm, format := frame.Waveform, frame.Format
	if w <= 0 || h <= 0 {
		return ""
	}
	if pcm.Count <= 0 || pcm.Count > visualizer.WaveformPoints ||
		format.SampleRate <= 0 || format.Channels < 1 || format.Channels > 2 {
		return fitBlock("Waiting for system audio PCM…", w, h)
	}
	metadata := fmt.Sprintf("PCM · system output · %d Hz · %d ch · %d frames · %.1f ms",
		format.SampleRate, format.Channels, pcm.Frames,
		1000*float64(pcm.Frames)/float64(format.SampleRate))
	lines := []string{metadata}
	cols := max(1, w-2)
	for ch := 0; ch < format.Channels; ch++ {
		rows := (h - 1) / format.Channels
		if rows < 1 {
			break
		}
		grid := make([][]rune, rows)
		for r := range grid {
			grid[r] = []rune(strings.Repeat(" ", cols))
		}
		toRow := func(v float32) int {
			return int(math.Round((1 - max(-1, min(1, float64(v)))) * float64(rows-1) / 2))
		}
		for x := 0; x < cols; x++ {
			lo := x * pcm.Count / cols
			hi := min(pcm.Count, max(lo+1, (x+1)*pcm.Count/cols))
			low, high := pcm.Points[lo].Min[ch], pcm.Points[lo].Max[ch]
			for i := lo + 1; i < hi; i++ {
				low, high = min(low, pcm.Points[i].Min[ch]), max(high, pcm.Points[i].Max[ch])
			}
			grid[toRow(0)][x] = '─'
			top, bottom := toRow(high), toRow(low)
			for r := top; r <= bottom; r++ {
				grid[r][x] = '│'
			}
			if top == bottom {
				grid[top][x] = '•'
			}
		}
		label := "M"
		if format.Channels == 2 {
			label = []string{"L", "R"}[ch]
		}
		for r := range grid {
			prefix := "  "
			if r == toRow(0) {
				prefix = label + " "
			}
			lines = append(lines, prefix+string(grid[r]))
		}
	}
	return fitBlock(strings.Join(lines, "\n"), w, h)
}
