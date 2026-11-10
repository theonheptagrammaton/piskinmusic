package main

import (
	"image"
	"reflect"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

const artworkCacheEntries = 16
const artworkCacheCells = 32768

// Event-loop owned, shared across the value copies made by Bubble Tea. Images
// and cached rows are immutable after publication. Limit both count and area so
// track changes and repeated resizes cannot accumulate terminal-sized strings.
type renderCache struct {
	art         []renderedArtwork
	cells       int
	geometry    layout
	geometryKey layoutKey
	hasGeometry bool
	lastFrame   string
	headerKey   headerKey
	header      string
}

type headerKey struct {
	w                                int
	screen                           screen
	bright, dim, selected, selection lipgloss.Color
	profile                          termenv.Profile
}

type renderedArtwork struct {
	img     image.Image
	w, h    int
	profile termenv.Profile
	rows    []string
}

type layoutKey struct {
	w, h              int
	screen            screen
	queue, eq, recent bool
}

func (m model) artworkRows(img image.Image, w, h int) []string {
	c := m.renderCache
	// An image implemented by a non-comparable value cannot be an identity key.
	if c == nil || img == nil || !reflect.ValueOf(img).Comparable() || w <= 0 || h <= 0 || w > artworkCacheCells/h {
		return renderArtwork(img, w, h)
	}
	profile := lipgloss.ColorProfile()
	for _, entry := range c.art {
		if entry.img == img && entry.w == w && entry.h == h && entry.profile == profile {
			return entry.rows
		}
	}
	rows := renderArtwork(img, w, h)
	for len(c.art) > 0 && (len(c.art) >= artworkCacheEntries || c.cells+w*h > artworkCacheCells) {
		c.cells -= c.art[0].w * c.art[0].h
		copy(c.art, c.art[1:])
		c.art[len(c.art)-1] = renderedArtwork{}
		c.art = c.art[:len(c.art)-1]
	}
	c.art = append(c.art, renderedArtwork{img, w, h, profile, rows})
	c.cells += w * h
	return rows
}
