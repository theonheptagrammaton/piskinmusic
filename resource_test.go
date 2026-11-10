package main

import (
	"image"
	"image/color"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/k1y0miiii/applemusic-tui/engine"
	"github.com/k1y0miiii/applemusic-tui/lastfm"
	"github.com/muesli/termenv"
)

type countedImage struct {
	image.Image
	reads int
}

func (img *countedImage) At(x, y int) color.Color {
	img.reads++
	return img.Image.At(x, y)
}

func TestArtworkCacheMatchesRendererAndBoundsResizes(t *testing.T) {
	previous := lipgloss.ColorProfile()
	defer lipgloss.SetColorProfile(previous)
	img := &countedImage{Image: image.NewUniform(color.RGBA{255, 80, 30, 255})}
	m := model{renderCache: &renderCache{}}
	for _, profile := range []termenv.Profile{termenv.TrueColor, termenv.ANSI256, termenv.Ascii} {
		lipgloss.SetColorProfile(profile)
		for _, size := range [][2]int{{24, 12}, {48, 24}, {12, 6}} {
			w, h := size[0], size[1]
			want := renderArtwork(img, w, h)
			got := m.artworkRows(img, w, h)
			if !slices.Equal(got, want) {
				t.Fatal("cached artwork differs from the original renderer")
			}
			reads := img.reads
			_ = m.artworkRows(img, w, h)
			if img.reads != reads {
				t.Fatal("warm cache resampled artwork")
			}
		}
	}
	for i := 1; i <= 100; i++ {
		_ = m.artworkRows(image.NewUniform(color.Black), i*2, i)
		if len(m.renderCache.art) > artworkCacheEntries || m.renderCache.cells > artworkCacheCells {
			t.Fatal("resize/track churn exceeded artwork cache bounds")
		}
	}
	other := image.NewUniform(color.White)
	if slices.Equal(m.artworkRows(other, 10, 5), m.artworkRows(img, 10, 5)) && lipgloss.ColorProfile() != termenv.Ascii {
		t.Fatal("replacement image reused stale artwork")
	}
}

func TestCachedLayoutAndHeaderFollowGeometryAndTheme(t *testing.T) {
	m := mouseModel(120, 40, 20)
	m.renderCache = &renderCache{}
	for _, size := range [][2]int{{40, 12}, {80, 24}, {120, 40}, {30, 8}} {
		m.w, m.h = size[0], size[1]
		for s := screenLibrary; s <= screenLyrics; s++ {
			m.screen = s
			for _, overlay := range []string{"none", "queue", "eq"} {
				m.queueOpen, m.eqOpen = overlay == "queue", overlay == "eq"
				for _, pos := range []time.Duration{time.Second, 12 * time.Minute, time.Hour} {
					m.st.Pos, m.st.Volume, m.st.Shuffle, m.st.Repeat = pos, int(pos.Seconds())%101, true, 2
					got := m.layout()
					uncached := m
					uncached.renderCache = nil
					if !reflect.DeepEqual(got, uncached.layout()) || m.header(got) != uncached.header(got) {
						t.Fatalf("cache changed rendering/input geometry: %v screen%d %s", size, s, overlay)
					}
				}
			}
		}
	}
	previous := themes[0]
	defer applyTheme(previous)
	for _, theme := range themes {
		applyTheme(theme)
		l := m.layout()
		uncached := m
		uncached.renderCache = nil
		if m.header(l) != uncached.header(l) {
			t.Fatal("header ignored theme change")
		}
	}
}

func TestAdaptiveCadenceAndHiddenMotion(t *testing.T) {
	m := performanceModel(screenVisualizer, true, false)
	if m.tickInterval() != time.Second/30 {
		t.Fatal("active visualization lost 30 Hz cadence")
	}
	m.cfg["ui.reduced_motion"] = "true"
	if m.tickInterval() != 100*time.Millisecond || m.pulseAmount() != 0 {
		t.Fatal("reduced motion did not lower cadence and disable pulse")
	}
	next, _ := m.Update(tickMsg(time.Now()))
	if next.(model).orbSpin != m.orbSpin || next.(model).orbKick != 0 {
		t.Fatal("reduced motion moved the shape")
	}
	m.cfg["ui.reduced_motion"] = "false"
	for _, hide := range []func(*model){
		func(m *model) { m.screen = screenLibrary },
		func(m *model) { m.queueOpen = true },
		func(m *model) { m.eqOpen = true },
		func(m *model) { m.helpOpen = true },
		func(m *model) { m.searchOpen = true },
		func(m *model) { m.acctOpen = true },
		func(m *model) { m.blurred = true },
		func(m *model) { m.w = 30 },
		func(m *model) { m.st.Playing = false },
	} {
		hidden := m
		hide(&hidden)
		if hidden.tickInterval() != 250*time.Millisecond {
			t.Fatal("hidden or idle screen kept fast cadence")
		}
		next, _ := hidden.Update(tickMsg(time.Now()))
		got := next.(model)
		if got.orbSpin != hidden.orbSpin || got.orbWobble != hidden.orbWobble {
			t.Fatal("hidden or paused shape advanced")
		}
		if hidden.st.Playing && got.wv == hidden.wv {
			t.Fatal("hidden visualization stopped live loudness history")
		}
	}
}

func TestBlurFreezesRenderingAndFocusRefreshes(t *testing.T) {
	m := performanceModel(screenFocus, true, false)
	img := &countedImage{Image: m.art}
	m.art = img
	frame := m.View()
	reads := img.reads
	next, _ := m.Update(tea.BlurMsg{})
	m = next.(model)
	m.st.Now.Title = "Changed while hidden"
	m.art = &countedImage{Image: image.NewUniform(color.White)}
	if m.View() != frame || img.reads != reads {
		t.Fatal("blurred terminal rendered new output")
	}
	next, _ = m.Update(tea.FocusMsg{})
	m = next.(model)
	if !strings.Contains(m.View(), "Changed while hidden") {
		t.Fatal("focus did not refresh current state")
	}
}

func TestAdaptiveTicksPreserveElapsedTimeAndCapSuspendListening(t *testing.T) {
	start := time.Unix(1700000000, 0)
	m := model{phase: phaseReady, lastTick: start, note: "notice",
		st: engine.State{Playing: true, Pos: time.Second, Dur: time.Hour,
			Now: engine.Track{Title: "Fixture", Artist: "Fixture", Duration: time.Hour}},
		scrobbler: &lastfm.Client{}, scrKey: "Fixture\x00Fixture",
	}
	for i := 1; i <= 28; i++ {
		next, _ := m.Update(tickMsg(start.Add(time.Duration(i) * 250 * time.Millisecond)))
		m = next.(model)
	}
	if m.t != 7 || m.st.Pos != 8*time.Second || m.scrPlayed != 7*time.Second || m.note != "" {
		t.Fatalf("adaptive ticks changed elapsed time: t=%v pos=%v listening=%v notice=%q", m.t, m.st.Pos, m.scrPlayed, m.note)
	}
	next, _ := m.Update(tickMsg(start.Add(time.Hour)))
	m = next.(model)
	if m.t != 3600 || m.scrPlayed != 8*time.Second || m.st.Pos != 9*time.Second {
		t.Fatal("suspend credited unobserved listening or slowed wall-time expiry")
	}
}

func TestHiddenPrimaryContentDoesNotSampleArtwork(t *testing.T) {
	for _, screen := range []screen{screenLibrary, screenVisualizer, screenFocus, screenLyrics} {
		m := performanceModel(screen, true, false)
		img := &countedImage{Image: m.art}
		m.art = img
		m.helpOpen = true
		_ = m.View()
		if img.reads != 0 {
			t.Fatal("blocking overlay rendered hidden artwork")
		}
	}
}
