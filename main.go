package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	audioInitializingText     = "initializing Apple Music audio…"
	audioInitializingWarning  = "Apple Music is still initializing · R reload"
	visualizerUnavailableNote = "visualizer unavailable · simulated"
	visualizerFrameFreshness  = 400 * time.Millisecond
)

// The live palette. Mutable because themes swap it at runtime; safe because
// Bubble Tea runs Update and View on one goroutine.
var (
	accent    lipgloss.Color
	accentHi  lipgloss.Color
	accentLo  lipgloss.Color
	fgBright  lipgloss.Color
	fgDim     lipgloss.Color
	fgFaint   lipgloss.Color
	borderDim lipgloss.Color
	selBg     lipgloss.Color
)

func init() { applyTheme(themes[0]) }

// version is stamped at build time with -ldflags "-X main.version=...".
// Release builds carry the tag; a plain `go build` stays "dev".
var version = "dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v", "version":
			fmt.Printf("amtui %s %s/%s\n", version, runtime.GOOS, runtime.GOARCH)
			return
		case "lastfm-auth":
			os.Exit(runLastfmAuth())
		}
	}
	m := model{
		statusCh:  make(chan string, 8),
		status:    "starting…",
		mprisQuit: make(chan struct{}, 1),
	}
	m.cfg = loadConfig()
	m.vizMode = loadVizMode()
	m.eqGains = loadEQGains()
	m.scrobbler = newScrobbler(m.cfg)
	m.artCache = newArtCache(8)
	t := themeFromConfig(m.cfg, loadThemeName())
	m.themeName = t.name
	applyTheme(t)
	// Cell motion rather than all motion: the app only reacts to clicks and the
	// wheel, and all-motion floods the loop with a message per pointer move.
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		if runtime.GOOS == "windows" {
			log.Fatalf("%v (amtui needs a Windows console — run it in Windows Terminal or PowerShell; from Git Bash/mintty: winpty amtui)", err)
		}
		log.Fatal(err)
	}
}
