package main

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func (m model) audioInitializingLine() string {
	sp := []rune("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")
	return lipgloss.NewStyle().Foreground(accentHi).Render(" "+string(sp[int(m.t*12)%len(sp)])+" ") +
		lipgloss.NewStyle().Foreground(fgDim).Render(audioInitializingText)
}

// pad truncates/pads a styled line to exactly w columns.
func pad(s string, w int) string {
	if w <= 0 {
		return ""
	}
	s = ansi.Truncate(s, w, "")
	return s + strings.Repeat(" ", max(0, w-lipgloss.Width(s)))
}

func centeredRows(s string, w, rows int) string {
	var b strings.Builder
	for r := 0; r < rows; r++ {
		line := ""
		if r == rows/2 {
			line = strings.Repeat(" ", max(0, (w-lipgloss.Width(s))/2)) + s
		}
		b.WriteString(pad(line, w) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// panel draws a bordered box. accentNow is the (possibly bass-pulsed) accent
// used for the focused border; callers pass m.pulsedAccent().
func panel(title, body string, w, h int, focused bool, accentNow lipgloss.Color) string {
	bc, tc := borderDim, fgFaint
	if focused {
		bc, tc = accentNow, accentHi
		title = "[" + title + "]"
	}
	head := lipgloss.NewStyle().Foreground(tc).Bold(true).Render(" " + title)
	content := pad(head, w) + "\n" + body
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).BorderForeground(bc).
		Width(w).Height(h).Render(content)
}

func (m model) queuePanel(w, h int) string {
	var b strings.Builder
	rows := h - 1
	if m.audioInitializing() {
		return centeredRows(m.audioInitializingLine(), w, rows)
	}
	if m.loading != "" && m.loadHideQueue {
		sp := []rune("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")
		line := lipgloss.NewStyle().Foreground(accentHi).Render(" "+string(sp[int(m.t*12)%len(sp)])+" ") +
			lipgloss.NewStyle().Foreground(fgDim).Render("loading ") +
			lipgloss.NewStyle().Foreground(fgBright).Bold(true).Render(m.loading) +
			lipgloss.NewStyle().Foreground(fgDim).Render("…")
		for r := 0; r < rows; r++ {
			s := ""
			if r == rows/2 {
				s = line
			}
			b.WriteString(pad(s, w) + "\n")
		}
		return strings.TrimRight(b.String(), "\n")
	}
	q := m.st.Queue
	if len(q) == 0 {
		empty := lipgloss.NewStyle().Foreground(fgFaint).Render(" queue is empty — press / to search")
		return pad(empty, w)
	}
	start := max(0, min(m.selIdx-rows/2, len(q)-rows))
	for r := 0; r < rows; r++ {
		i := start + r
		if i >= len(q) {
			b.WriteString(pad("", w) + "\n")
			continue
		}
		tr := q[i]
		prefix, ts, as := "  ", fgDim, fgFaint
		if i == m.st.QueuePos {
			prefix, ts, as = "▶ ", fgBright, fgDim
		}
		if i == m.selIdx {
			if i == m.st.QueuePos {
				prefix = "▸▶"
			} else {
				prefix = "▸ "
			}
		}
		left := lipgloss.NewStyle().Foreground(accentHi).Render(prefix) +
			lipgloss.NewStyle().Foreground(ts).Render(tr.Title) +
			lipgloss.NewStyle().Foreground(as).Render(" — "+tr.Artist)
		dur := lipgloss.NewStyle().Foreground(fgFaint).Render(fmtTime(tr.Duration))
		gap := w - lipgloss.Width(left) - lipgloss.Width(dur) - 1
		line := left + strings.Repeat(" ", max(1, gap)) + dur
		if m.focus == focusQueue && i == m.selIdx {
			line = selectedRow(prefix+tr.Title+" — "+tr.Artist+" "+fmtTime(tr.Duration), w)
		}
		b.WriteString(pad(line, w) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// recentTitle counts the grid rows so the scroll position is visible.
func (m model) recentTitle() string {
	if len(m.recent) == 0 {
		return "RECENTLY PLAYED"
	}
	rows := (len(m.recent) + recentCols - 1) / recentCols
	return fmt.Sprintf("RECENTLY PLAYED · %d/%d", m.recentSel/recentCols+1, rows)
}

// placeholderTile stands in until a cover arrives, or forever when the entry has
// no artwork at all.
func placeholderTile(w, h int) []string {
	st := lipgloss.NewStyle().Foreground(borderDim)
	out := make([]string, h)
	for r := range out {
		out[r] = st.Render(strings.Repeat("░", w))
	}
	return out
}

// recentPanel draws one row of cover tiles from the recently-played list.
// Rows scroll with ↑/↓; ←/→ move between the tiles of the current row.
func (m model) recentPanel(w, h int) string {
	return m.renderRecentPanel(w, h, recentGrid(w, h-1))
}

func (m model) renderRecentPanel(w, h int, gridLayout recentGridLayout) string {
	faint := lipgloss.NewStyle().Foreground(fgFaint)
	rows := h - 1
	if len(m.recent) == 0 {
		return centeredRows(faint.Render(" nothing played yet"), w, rows)
	}
	artCols, artRows, gap := gridLayout.artCols, gridLayout.artRows, gridLayout.gap
	if artCols == 0 {
		return centeredRows(faint.Render(" pane too narrow for covers"), w, rows)
	}
	tileW := artCols

	start := m.recentSel / recentCols * recentCols
	grid := make([]string, artRows+2)
	for c := 0; c < recentCols; c++ {
		for r := range grid {
			grid[r] += strings.Repeat(" ", gap)
		}
		i := start + c
		if i >= len(m.recent) {
			for r := range grid {
				grid[r] += strings.Repeat(" ", tileW)
			}
			continue
		}
		tr := m.recent[i]
		cover := placeholderTile(artCols, artRows)
		if img := m.tileArt[tr.ID]; img != nil && configBool(m.cfg, "artwork.enabled", true) {
			cover = m.artworkRows(img, artCols, artRows)
		}
		for r := 0; r < artRows; r++ {
			line := ""
			if r < len(cover) {
				line = cover[r]
			}
			grid[r] += pad(line, tileW)
		}
		ts, as, mark := lipgloss.NewStyle().Foreground(fgDim), faint, " "
		if i == m.recentSel {
			mark = "▸"
		}
		if m.focus == focusRecent && i == m.recentSel {
			ts = lipgloss.NewStyle().Foreground(accentHi).Bold(true)
			as = lipgloss.NewStyle().Foreground(fgDim)
			mark = "▸"
		}
		title := pad(ts.Render(mark+tr.Title), tileW)
		artist := pad(as.Render(" "+tr.Artist), tileW)
		if m.focus == focusRecent && i == m.recentSel {
			title, artist = selectedRow(mark+tr.Title, tileW), selectedRow(" "+tr.Artist, tileW)
		}
		grid[artRows] += title
		grid[artRows+1] += artist
	}
	for r := range grid {
		grid[r] = pad(grid[r], w)
	}
	for len(grid) < rows {
		grid = append(grid, pad("", w))
	}
	return strings.Join(grid[:rows], "\n")
}

func liveBarHeights(bands [32]float64, bars, rows int) []float64 {
	if bars <= 0 {
		return nil
	}
	heights := make([]float64, bars)
	for i := range heights {
		start := i * len(bands) / bars
		end := (i + 1) * len(bands) / bars
		if end <= start {
			end = start + 1
		}
		end = min(end, len(bands))

		var peak float64
		for _, value := range bands[start:end] {
			peak = max(peak, value)
		}
		heights[i] = min(peak*1.15, 1) * float64(max(rows, 0))
	}
	return heights
}

// peakDecay is the fall rate of the hold markers, per 30 fps frame.
const peakDecay = 0.012

// decayPeaks makes each peak follow its band up instantly and fall slowly —
// the Winamp behaviour.
func decayPeaks(peaks *[32]float64, bands [32]float64) {
	for i := range peaks {
		if bands[i] > peaks[i] {
			peaks[i] = bands[i]
			continue
		}
		peaks[i] = max(peaks[i]-peakDecay, 0)
	}
}

// waveBuckets is the resolution of the recorded loudness timeline. Wider than any
// realistic progress bar, so a resize never loses detail.
const waveBuckets = 240

// wave records band-derived loudness along the track for the progress bar.
// This timeline is independent of the time-domain PCM envelope.
type wave struct {
	samples [waveBuckets]float64
}

// record stores level at position frac (0..1), keeping the loudest value seen
// in that bucket.
func (w *wave) record(frac, level float64) {
	i := int(frac * waveBuckets)
	i = min(max(i, 0), waveBuckets-1)
	if level > w.samples[i] {
		w.samples[i] = level
	}
}

func (w *wave) reset() { w.samples = [waveBuckets]float64{} }

// column returns the peak level for column col of a bar cols wide.
func (w *wave) column(col, cols int) float64 {
	if cols <= 0 {
		return 0
	}
	start := col * waveBuckets / cols
	end := (col + 1) * waveBuckets / cols
	end = min(max(end, start+1), waveBuckets)
	peak := 0.0
	for _, v := range w.samples[start:end] {
		peak = max(peak, v)
	}
	return peak
}

// bandsLevel is the overall loudness: the mean of all bands.
func bandsLevel(bands [32]float64) float64 {
	sum := 0.0
	for _, v := range bands {
		sum += v
	}
	return sum / float64(len(bands))
}

// bassLevel is the mean of the four lowest bands — what drives the pulse.
func bassLevel(bands [32]float64) float64 {
	sum := 0.0
	for _, v := range bands[:4] {
		sum += v
	}
	return sum / 4
}

// pulseAmount is the current bass energy, or 0 when the pulse is switched off.
func (m model) pulseAmount() float64 {
	if m.reducedMotion() || m.blurred || !configBool(m.cfg, "visualizer.pulse", true) || !m.st.Playing {
		return 0
	}
	return bassLevel(m.vizBands)
}

func (m model) pulsedAccent() lipgloss.Color { return pulse(accent, m.pulseAmount()) }

// simulatedBands is the explicit compatibility demo animation. It is never PCM.
func simulatedBands(t float64, playing bool) [32]float64 {
	var bands [32]float64
	for i := range bands {
		x := t*1.35 + float64(i)*0.55
		a := 0.24 + 0.62*math.Abs(math.Sin(x))*(0.5+0.5*math.Sin(t*0.43+float64(i)*1.9))
		if !playing {
			a *= 0.06 // ponytail: bars just collapse on pause, no decay animation
		}
		bands[i] = min(max(a, 0), 1)
	}
	return bands
}

// barZoneColor picks accentLo/accent/accentHi by how high up the bar area a
// row sits. Shared by the visualizer and the EQ panel so the two can't drift
// apart on this.
func barZoneColor(level float64, rows int) lipgloss.Color {
	zone := level / float64(rows)
	switch {
	case zone > 0.66:
		return accentHi
	case zone > 0.33:
		return accent
	default:
		return accentLo
	}
}

type barCellKind int

const (
	barCellEmpty barCellKind = iota
	barCellFull
	barCellPartial
	barCellPeak
)

// barPartialGlyphs is the fractional top-of-bar cell, empty to full.
var barPartialGlyphs = []rune("▁▂▃▄▅▆▇█")

// barCellAt decides what one column draws at one row: full once its bar
// height reaches this row's level (counting rows from the bottom of the bar
// area), a fractional glyph for the row the bar's top lands in, a peak marker
// above that if one is tracked and lands here, otherwise empty. Shared by the
// visualizer and the EQ panel.
func barCellAt(height, level, peak float64, hasPeak bool) (kind barCellKind, glyph rune) {
	switch {
	case height >= level:
		return barCellFull, 0
	case height > level-1:
		return barCellPartial, barPartialGlyphs[min(7, int((height-level+1)*8))]
	case hasPeak && peak > level-1 && peak <= level:
		return barCellPeak, 0
	default:
		return barCellEmpty, 0
	}
}

func (m model) vizPanel(w, h int) string {
	if w <= 0 || h <= 0 {
		return ""
	}
	switch m.captureStatus() {
	case captureStarting:
		return fitBlock("Waiting for system audio PCM…", w, h)
	case captureStale:
		return fitBlock("System audio capture is stale", w, h)
	case captureUnavailable:
		return fitBlock("System audio capture unavailable", w, h)
	}
	switch m.vizMode {
	case vizWaveform:
		if m.captureStatus() == captureDemo {
			return fitBlock("PCM unavailable in demo", w, h)
		}
		return pcmPanel(m.vizPCM, w, h)
	case vizTorus:
		return orbPanel(w, h-1, m.orbSpin, m.orbWobble, m.vizBands)
	case vizSphere:
		return spherePanel(w, h-1, m.orbSpin, m.orbWobble, m.orbKick, m.vizBands)
	}
	rows, bars := h-1, max(1, w/3)
	heights := liveBarHeights(m.vizBands, bars, rows)
	var peaks []float64
	if configBool(m.cfg, "visualizer.peaks", true) {
		peaks = liveBarHeights(m.vizPeaks, bars, rows)
	}
	var b strings.Builder
	for r := 0; r < rows; r++ {
		level := float64(rows - r)
		st := lipgloss.NewStyle().Foreground(barZoneColor(level, rows))
		markSt := lipgloss.NewStyle().Foreground(fgDim)
		var row strings.Builder
		for i := 0; i < bars; i++ {
			peak := 0.0
			if peaks != nil {
				peak = peaks[i]
			}
			switch kind, glyph := barCellAt(heights[i], level, peak, peaks != nil); kind {
			case barCellFull:
				row.WriteString(st.Render("██"))
			case barCellPartial:
				c := string(glyph)
				row.WriteString(st.Render(c + c))
			case barCellPeak:
				row.WriteString(markSt.Render("▔▔"))
			default:
				row.WriteString("  ")
			}
			row.WriteString(" ")
		}
		b.WriteString(pad(" "+row.String(), w) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m model) lyricsPanel(w, h int) string {
	rows := h - 1
	artCols, artRows := 0, 0
	if configBool(m.cfg, "artwork.enabled", true) && m.art != nil {
		artCols, artRows = artworkLayout(w, rows, configInt(m.cfg, "artwork.min_lyrics_width", 30), m.lyricsWidth())
	}
	textW := w
	if artCols > 0 {
		textW = w - artCols - artworkGap
	}
	text := m.lyricsText(textW, rows)
	if artCols == 0 {
		return text
	}
	cover := m.artworkRows(m.art, artCols, artRows)
	lines := strings.Split(text, "\n")
	top := (rows - artRows) / 2
	var b strings.Builder
	for r := 0; r < rows; r++ {
		line := ""
		if r < len(lines) {
			line = lines[r]
		}
		// Center the cover vertically against the lyrics block.
		left := strings.Repeat(" ", artCols)
		if r >= top && r-top < len(cover) {
			left = cover[r-top]
		}
		b.WriteString(left + strings.Repeat(" ", artworkGap) + pad(line, textW) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// lyricsWidth is how many columns the lyrics actually use: the widest line plus
// the one-column indent lyricsText draws. Zero when there are no lines, which
// leaves the cover bounded by the configured minimum instead.
func (m model) lyricsWidth() int {
	n := 0
	for _, l := range m.ly.Lines {
		if w := lipgloss.Width(l.Text); w > n {
			n = w
		}
	}
	if n == 0 {
		return 0
	}
	return n + 1
}

// lyricsText is the text column of the lyrics panel, sized independently of the
// cover so the two can sit side by side.
func (m model) lyricsText(w, rows int) string {
	faint := lipgloss.NewStyle().Foreground(fgFaint)
	center := func(s string) string {
		var b strings.Builder
		for r := 0; r < rows; r++ {
			line := ""
			if r == rows/2 {
				line = s
			}
			b.WriteString(pad(line, w) + "\n")
		}
		return strings.TrimRight(b.String(), "\n")
	}
	switch {
	case m.lyBusy:
		sp := []rune("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")
		return center(faint.Render(" " + string(sp[int(m.t*12)%len(sp)]) + " looking up lyrics…"))
	case len(m.ly.Lines) == 0:
		return center(faint.Render(" no lyrics for this track"))
	}
	// pick the line to center on: by timestamp when synced, by progress when plain
	cur := -1
	if m.ly.Synced {
		cur = m.ly.Current(m.st.Pos)
	} else if m.st.Dur > 0 {
		cur = int(float64(m.st.Pos) / float64(m.st.Dur) * float64(len(m.ly.Lines)))
	}
	start := max(0, min(cur-rows/2, len(m.ly.Lines)-rows))
	var b strings.Builder
	for r := 0; r < rows; r++ {
		i := start + r
		if i >= len(m.ly.Lines) {
			b.WriteString(pad("", w) + "\n")
			continue
		}
		text := m.ly.Lines[i].Text
		st := faint // past lines
		if i > cur {
			st = lipgloss.NewStyle().Foreground(fgDim) // upcoming, slightly brighter
		}
		if m.ly.Synced && i == cur {
			st = lipgloss.NewStyle().Foreground(accentHi).Bold(true)
			if text == "" {
				text = "♪"
			}
		}
		b.WriteString(pad(st.Render(" "+text), w) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func fmtTime(d time.Duration) string {
	s := int(d.Seconds())
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}

// fmtLongTime is fmtTime with an hours field for queue totals.
func fmtLongTime(d time.Duration) string {
	s := int(d.Seconds())
	if s >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", s/3600, (s%3600)/60, s%60)
	}
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}

// queueTitle labels the queue panel with the position and the time left,
// counting the unplayed part of the current track plus everything after it.
func (m model) queueTitle() string {
	q := m.st.Queue
	if len(q) == 0 || m.st.QueuePos < 0 || m.st.QueuePos >= len(q) {
		return "QUEUE"
	}
	left := q[m.st.QueuePos].Duration - m.st.Pos
	if left < 0 {
		left = 0
	}
	for _, tr := range q[m.st.QueuePos+1:] {
		left += tr.Duration
	}
	return fmt.Sprintf("QUEUE · %d/%d · %s left",
		m.st.QueuePos+1, len(q), fmtLongTime(left))
}

func (m model) transportPanel(w int) string {
	_, barW := m.progressGeometry(w)
	return m.renderTransportPanel(w, barW)
}

func (m model) renderTransportPanel(w, barW int) string {
	if w < 68 {
		return m.compactTransport(w, barW)
	}
	icon := "⏸"
	if !m.st.Playing {
		icon = "▶"
	}
	dim := lipgloss.NewStyle().Foreground(fgDim)
	faint := lipgloss.NewStyle().Foreground(fgFaint)
	pink := lipgloss.NewStyle().Foreground(accentHi)

	title, artist := m.st.Now.Title, m.st.Now.Artist
	if title == "" {
		title, artist = "nothing playing", "press / to find music"
	}
	hints := transportHints(m.focus)
	if m.eqOpen {
		hints = eqHints
	}
	if m.note != "" {
		hints = m.note + " "
	}
	mark := " ▀▀ "
	if m.focus == focusPlayer {
		mark = " ▸  "
	}
	l1 := pink.Render(mark) +
		lipgloss.NewStyle().Foreground(fgBright).Bold(true).Render(title) +
		faint.Render(" — "+artist)
	if album := m.st.Now.Album; album != "" {
		l1 += faint.Render(" · " + album)
	}
	if m.audioInitializing() {
		l1 = m.audioInitializingLine()
	} else if m.loading != "" {
		sp := []rune("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")
		l1 = pink.Render(" "+string(sp[int(m.t*12)%len(sp)])+" ") +
			dim.Render("loading ") +
			lipgloss.NewStyle().Foreground(fgBright).Bold(true).Render(m.loading) +
			dim.Render("…")
	}
	// The hints share this row with the track title, so on a narrow terminal
	// there may be no room for them. pad() truncates silently, which is how the
	// old line lost its tail without anyone noticing; drop to the short form
	// instead, and then to nothing, so whatever is shown is shown whole.
	hints = fitHints(hints, w-lipgloss.Width(l1)-1)
	r1 := faint.Render(hints)
	line1 := l1 + strings.Repeat(" ", max(1, w-lipgloss.Width(l1)-lipgloss.Width(r1))) + r1

	volSegs := (m.st.Volume + 9) / 17 // 0..6 segments
	volSt := faint
	if m.focus == focusPlayer {
		volSt = pink
	}
	shuf, rep := "off", [3]string{"off", "one", "all"}[min(m.st.Repeat, 2)]
	if m.st.Shuffle {
		shuf = "on"
	}
	r2 := faint.Render(fmt.Sprintf("⇄ %s  ↻ %s  vol ", shuf, rep)) +
		volSt.Render(strings.Repeat("▮", volSegs)+strings.Repeat("▯", 6-volSegs)) +
		dim.Render(fmt.Sprintf(" %d%% ", m.st.Volume))
	times := fmtTime(m.st.Pos) + " "
	timee := " " + fmtTime(m.st.Dur)
	frac := 0.0
	if m.st.Dur > 0 {
		frac = min(float64(m.st.Pos)/float64(m.st.Dur), 1)
	}
	filled := int(frac * float64(barW))
	bar := m.progressBar(barW, filled)
	l2 := lipgloss.NewStyle().Foreground(accent).Render(" ▄▄ ") +
		pink.Render(icon+" ") + dim.Render(times) + bar + dim.Render(timee)
	line2 := l2 + strings.Repeat(" ", max(1, w-lipgloss.Width(l2)-lipgloss.Width(r2))) + r2
	return pad(line1, w) + "\n" + pad(line2, w)
}

// progressBar draws the played part as recorded band-derived loudness and
// the rest as a flat rule. Falls back to a plain filled bar when the timeline
// is disabled in the config.
func (m model) progressBar(barW, filled int) string {
	rest := lipgloss.NewStyle().Foreground(borderDim).
		Render(strings.Repeat("─", max(0, barW-filled)))
	if !configBool(m.cfg, "visualizer.wave", true) {
		return lipgloss.NewStyle().Foreground(pulse(accentHi, m.pulseAmount())).
			Render(strings.Repeat("━", filled)) + rest
	}
	steps := []rune("▁▂▃▄▅▆▇█")
	st := lipgloss.NewStyle().Foreground(pulse(accentHi, m.pulseAmount()))
	var b strings.Builder
	for c := 0; c < filled; c++ {
		lvl := m.wv.column(c, barW)
		b.WriteString(st.Render(string(steps[min(int(lvl*float64(len(steps))), len(steps)-1)])))
	}
	return b.String() + rest
}

func (m model) bootView() string {
	logo := lipgloss.NewStyle().Foreground(accentHi).Bold(true).Render("PISKIN MUSIC")
	sub := lipgloss.NewStyle().Foreground(fgFaint).Render("apple music in your terminal")
	var line string
	if m.phase == phaseFail {
		full := lipgloss.NewStyle().Foreground(accent).Render("✕ "+m.status) +
			lipgloss.NewStyle().Foreground(fgFaint).Render("  ·  q quit")
		// Wrap instead of letting the fixed-grid renderer clip long errors —
		// open()'s AMTUI_CHROME hint must stay readable on an 80-col terminal.
		line = lipgloss.NewStyle().Width(max(1, min(m.w-4, 76))).Render(full)
	} else {
		sp := []rune("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")
		line = lipgloss.NewStyle().Foreground(fgDim).
			Render(string(sp[int(m.t*12)%len(sp)]) + " " + m.status)
	}
	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center,
		logo+"\n"+sub+"\n\n"+line)
}

func (m model) View() string {
	if m.blurred && m.renderCache != nil && m.renderCache.lastFrame != "" {
		return m.renderCache.lastFrame
	}
	if m.w <= 0 || m.h <= 0 {
		return ""
	}
	lay := m.layout()
	var frame string
	switch {
	case m.helpOpen:
		frame = m.helpView()
	case m.acctOpen:
		frame = m.accountView()
	case m.searchOpen:
		frame = m.searchView()
	case m.phase != phaseReady:
		frame = m.bootView()
	case !lay.visible:
		frame = "PISKIN MUSIC\n" + m.st.Now.Title + "\nterminal too small · need 40×12\nspace pause · ? help · q quit"
	default:
		var left string
		if lay.queueBody.h > 0 {
			title := m.queueTitle()
			if m.queueOpen {
				title += " · c / esc close"
			}
			left = panel(title, m.queuePanel(lay.qw, lay.qh), lay.qw, lay.qh, m.focus == focusQueue, m.pulsedAccent())
			if lay.recH > 0 {
				left = lipgloss.JoinVertical(lipgloss.Left, left, panel(m.recentTitle(), m.renderRecentPanel(lay.qw, lay.recH-2, lay.recentGrid), lay.qw, lay.recH-2, m.focus == focusRecent, m.pulsedAccent()))
			}
		}
		var main string
		if lay.main.w > 0 {
			w, h := lay.main.w-2, lay.main.h-2
			title, body := "", ""
			switch {
			case m.eqOpen:
				title, body = eqTitle(m.eqGains, m.eqSel), m.eqBody(w, h, lay)
			case m.screen == screenLibrary:
				title, body = "LIBRARY · [ / ] category", m.libraryPanel(w, h)
			case m.screen == screenFocus:
				title, body = "NOW PLAYING", m.focusPanel(w, h)
			case m.screen == screenVisualizer:
				title, body = m.visualizerTitle()+" · "+vizModeName(m.vizMode), m.vizPanel(w, h)
			case m.screen == screenLyrics:
				title, body = "LYRICS", m.lyricsPanel(w, h)
			}
			main = panel(title, body, w, h, m.focus == focusLibrary, m.pulsedAccent())
		}
		top := lipgloss.JoinHorizontal(lipgloss.Top, left, main)
		transport := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(map[bool]lipgloss.Color{true: m.pulsedAccent(), false: borderDim}[m.focus == focusPlayer]).Width(lay.transport.w).Render(m.renderTransportPanel(lay.transport.w, lay.progress.w))
		frame = m.header(lay) + "\n" + top + "\n" + transport
	}
	frame = lipgloss.NewStyle().Background(background).Foreground(fgBright).Render(fitBlock(frame, m.w, m.h))
	if m.renderCache != nil {
		m.renderCache.lastFrame = frame
	}
	return frame
}

func (m model) visualizerTitle() string {
	switch m.captureStatus() {
	case captureLive:
		return "VISUALIZER · LIVE · " + m.vizSource
	case captureStarting:
		return "VISUALIZER · STARTING"
	case captureStale:
		return "VISUALIZER · STALE · " + m.vizSource
	case captureDemo:
		return "VISUALIZER · DEMO · SIMULATED"
	default:
		return "VISUALIZER · UNAVAILABLE"
	}
}
