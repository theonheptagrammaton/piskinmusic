package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	audioInitializingText    = "initializing Apple Music audio…"
	audioInitializingWarning = "Apple Music is still initializing · R reload"
	visualizerFrameFreshness = 400 * time.Millisecond
)

func init() { applyTheme(themes[0]) }

// version is stamped at build time with -ldflags "-X main.version=...".
// Release builds carry the tag; a plain `go build` stays "dev".
var version = "dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v", "version":
			fmt.Printf("PISKIN MUSIC (amtui) %s %s/%s\n", version, runtime.GOOS, runtime.GOARCH)
			return
		case "lastfm-auth":
			os.Exit(runLastfmAuth())
		}
	}
	m := model{
		statusCh:    make(chan string, 8),
		status:      "starting…",
		focus:       focusLibrary,
		screenFocus: [4]int{focusLibrary, focusPlayer, focusPlayer, focusPlayer},
		mprisQuit:   make(chan struct{}, 1),
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
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion(), tea.WithReportFocus())
	if _, err := p.Run(); err != nil {
		if runtime.GOOS == "windows" {
			log.Fatalf("%v (amtui needs a Windows console — run it in Windows Terminal or PowerShell; from Git Bash/mintty: winpty amtui)", err)
		}
		log.Fatal(err)
	}
}
