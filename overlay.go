package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/k1y0miiii/applemusic-tui/engine"
)

var tabNames = [4]string{"RECENT", "SONGS", "ALBUMS", "PLAYLISTS"}

func (m model) curList() []engine.Track {
	switch m.sTab {
	case 1:
		return m.sRes.Songs
	case 2:
		return m.sRes.Albums
	case 3:
		return m.sRes.Playlists
	}
	return m.sRes.Recent
}

func (m model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	eng := m.eng
	list := m.curList()
	switch msg.String() {
	case "esc", "ctrl+c":
		m.searchOpen = false
	case "enter":
		if m.sInput {
			if strings.TrimSpace(m.sQuery) != "" {
				m.sBusy = true
				return m, m.searchCmd()
			}
			m.sBusy = true // empty query = reload library
			return m, m.libraryCmd()
		} else if m.sSel < len(list) {
			if m.audioInitializing() {
				m.note, m.noteAt = audioInitializingText, m.t
				return m, nil
			}
			kind, id, title := list[m.sSel].Kind, list[m.sSel].ID, list[m.sSel].Title
			m.searchOpen = false
			m.loading, m.loadSnap, m.loadStart = title, snap(m.st), m.t
			m.loadHideQueue = true // the whole queue is being replaced
			m.beginAudioInitialization()
			return m, doCmd(func() error { return eng.Play(kind, id) })
		}
	case "tab":
		m.sTab, m.sSel = (m.sTab+1)%4, 0
	case "/":
		m.sInput = true
	case "backspace":
		m.sInput = true
		if len(m.sQuery) > 0 {
			r := []rune(m.sQuery)
			m.sQuery = string(r[:len(r)-1])
		}
	case "up":
		if m.sInput {
			m.sInput = false // arrow dives from input into the list
		} else {
			m.sSel = max(m.sSel-1, 0)
		}
	case "down":
		if m.sInput {
			m.sInput, m.sSel = false, 0
		} else {
			m.sSel = min(m.sSel+1, max(0, len(list)-1))
		}
	case "k":
		if m.sInput {
			m.sQuery += "k"
		} else {
			m.sSel = max(m.sSel-1, 0)
		}
	case "j":
		if m.sInput {
			m.sQuery += "j"
		} else {
			m.sSel = min(m.sSel+1, max(0, len(list)-1))
		}
	case "a", "A":
		if !m.sInput {
			if m.sSel < len(list) {
				kind, id := list[m.sSel].Kind, list[m.sSel].ID
				f := func(kind, id string) error { return eng.QueueLater(kind, id) }
				m.note, m.noteAt = "added to queue: "+list[m.sSel].Title, m.t
				if msg.String() == "A" {
					f = func(kind, id string) error { return eng.QueueNext(kind, id) }
					m.note = "playing next: " + list[m.sSel].Title
				}
				return m, doCmd(func() error { return f(kind, id) })
			}
		} else {
			m.sQuery += msg.String()
		}
	default:
		if msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace {
			m.sInput = true // typing anywhere returns focus to the input
			m.sQuery += string(msg.Runes)
			if msg.Type == tea.KeySpace {
				m.sQuery += " "
			}
		}
	}
	return m, nil
}

func (m model) searchGeometry() (rect, []keyTarget, []keyTarget) {
	body := rect{1, 4, max(0, m.w-2), max(0, m.h-6)}
	var tabs, actions []keyTarget
	x := 1
	for i, name := range tabNames {
		tabs = append(tabs, keyTarget{rect{x, 2, len(name) + 2, 1}, fmt.Sprint(i), " " + name + " "})
		x += len(name) + 2
	}
	x = 1
	for _, a := range []struct{ key, label string }{{"enter", "↵ Play/Search"}, {"a", "a Queue"}, {"A", "A Next"}, {"esc", "× Close"}} {
		actions = append(actions, keyTarget{rect{x, m.h - 2, lipgloss.Width(a.label), 1}, a.key, a.label})
		x += lipgloss.Width(a.label) + 1
	}
	return body.clipped(m.w, m.h), clipTargets(tabs, m.w, m.h), clipTargets(actions, m.w, m.h)
}
func (m model) searchView() string {
	body, tabs, actions := m.searchGeometry()
	cursor := ""
	if m.sInput {
		cursor = "▏"
	}
	lines := []string{pad("SEARCH APPLE MUSIC", m.w), pad(" / "+m.sQuery+cursor, m.w)}
	tabLine := " "
	for i, t := range tabs {
		label := t.label
		if i == m.sTab {
			label = selectedRow("["+tabNames[i]+"]", t.w)
		}
		tabLine += label
	}
	lines = append(lines, pad(tabLine, m.w), pad("", m.w))
	list := m.curList()
	start := listWindow(m.sSel, len(list), body.h)
	for r := 0; r < body.h; r++ {
		line := ""
		i := start + r
		if m.sBusy {
			if r == 0 {
				line = " searching…"
			}
		} else if i < len(list) {
			tr := list[i]
			line = " " + tr.Title + " — " + tr.Artist
			if tr.Duration > 0 {
				line += " " + fmtTime(tr.Duration)
			}
			if !m.sInput && i == m.sSel {
				line = selectedRow("▸"+line, body.w)
			}
		} else if r == 0 {
			line = " no results · type then enter to search"
		}
		lines = append(lines, pad(" "+pad(line, body.w), m.w))
	}
	footer := " "
	for _, a := range actions {
		footer += a.label + " "
	}
	lines = append(lines, pad(footer, m.w), pad(m.note, m.w))
	return strings.Join(lines, "\n")
}
