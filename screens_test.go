package main

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/k1y0miiii/applemusic-tui/engine"
)

func TestScreenNavigationPreservesPlaybackAndSelections(t *testing.T) {
	m := recentModel()
	m.eng = &commandPlayback{}
	m.selIdx, m.recentSel, m.librarySel, m.sSel = 2, 3, [3]int{4, 5, 6}, 7
	m.sQuery = "retained query"
	m.sRes.Songs = []engine.Track{{ID: "retained result"}}
	before := m
	for i := 0; i < 4; i++ {
		keyboard, cmd := m.Update(keyMessage(fmt.Sprint(i + 1)))
		if cmd != nil {
			t.Fatal("screen navigation performed effects")
		}
		l := m.layout()
		pointer := click(m, l.nav[i].x, l.nav[i].y)
		m = keyboard.(model)
		if m.screen != screen(i) || pointer.screen != m.screen || pointer.focus != m.focus {
			t.Fatalf("screen %d keyboard/mouse differ", i)
		}
		if !reflect.DeepEqual(m.st, before.st) || m.selIdx != before.selIdx || m.recentSel != before.recentSel || m.librarySel != before.librarySel || m.sSel != before.sSel || m.sQuery != before.sQuery || !reflect.DeepEqual(m.sRes, before.sRes) {
			t.Fatal("navigation lost playback or selection")
		}
		if i != 0 && m.focus == focusRecent {
			t.Fatal("hidden recent grid focused")
		}
	}
}

func TestScreenShortcutsRespectTextAndBlockingOverlays(t *testing.T) {
	for _, kind := range []string{"search", "EQ", "queue", "account", "help"} {
		m := recentModel()
		switch kind {
		case "search":
			m.searchOpen, m.sInput = true, true
		case "EQ":
			m.eqOpen = true
			next, cmd := m.Update(keyMessage("enter"))
			if cmd != nil || next.(model).loading != "" {
				t.Fatal("EQ enter played a hidden selection")
			}

		case "queue":
			m.toggleQueue()
		case "account":
			m.acctOpen = true
		case "help":
			m.helpOpen = true
		}
		for _, digit := range []string{"1", "2", "3", "4"} {
			next, _ := m.Update(keyMessage(digit))
			m = next.(model)
			if m.screen != screenLibrary {
				t.Fatalf("%s allowed screen shortcut", kind)
			}
			// Help/account dismiss on a key, so reopen for the next ownership check.
			if kind == "account" {
				m.acctOpen = true
			}
			if kind == "help" {
				m.helpOpen = true
			}
		}
		if kind == "search" && m.sQuery != "1234" {
			t.Fatalf("screen shortcuts ate query: %q", m.sQuery)
		}
	}
}

func TestFocusTraversalUsesOnlyVisibleRegionsInBothDirections(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {60, 20}, {80, 24}, {120, 32}, {180, 50}} {
		for s := screenLibrary; s <= screenLyrics; s++ {
			m := recentModel()
			m.w, m.h, m.screen = size[0], size[1], s
			m.normalizeFocus()
			start := m.focus
			zones := m.focusZones()
			visited := map[int]bool{}
			for range zones {
				visited[m.focus] = true
				next, _ := m.Update(keyMessage("tab"))
				m = next.(model)
			}
			if len(visited) != len(zones) || m.focus != start {
				t.Fatalf("%v %v bad traversal %v", size, s, visited)
			}
			for range zones {
				next, _ := m.Update(keyMessage("shift+tab"))
				m = next.(model)
			}
			if m.focus != start {
				t.Fatal("reverse traversal failed to wrap")
			}
			for i := 0; i < 3; i++ {
				next, _ := m.Update(keyMessage("tab"))
				m = next.(model)
			}
			saved := m.focus
			other := (s + 1) % 4
			m.selectScreen(other)
			m.selectScreen(s)
			if m.focus != saved {
				t.Fatal("screen focus was not retained")
			}
		}
	}
}

func TestQueueOverlayAvailableOnEveryScreenAndSize(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {60, 20}, {120, 32}, {180, 50}} {
		for s := screenLibrary; s <= screenLyrics; s++ {
			m := recentModel()
			m.w, m.h, m.screen, m.focus = size[0], size[1], s, focusPlayer
			next, cmd := m.Update(keyMessage("c"))
			m = next.(model)
			if cmd != nil || !m.queueOpen || m.layout().queueBody.w != m.w-2 || !strings.Contains(m.View(), m.st.Queue[0].Title) {
				t.Fatalf("queue not reachable at %v on %d", size, s)
			}
			next, _ = m.Update(keyMessage("esc"))
			m = next.(model)
			if m.queueOpen || m.screen != s || m.focus != focusPlayer {
				t.Fatal("closing queue lost context")
			}
		}
	}
}

func TestLibrarySnapshotSurvivesCatalogSearch(t *testing.T) {
	m := recentModel()
	res := engine.SearchResults{Albums: demoRecent(20), Playlists: demoRecent(3), Recent: demoRecent(7)}
	next, _ := m.Update(searchMsg{res: res, keepInput: true})
	m = next.(model)
	m.librarySel = [3]int{12, 2, 4}
	next, _ = m.Update(searchMsg{res: engine.SearchResults{Songs: demoRecent(2)}})
	m = next.(model)
	if !reflect.DeepEqual(m.library, res) || m.librarySel != [3]int{12, 2, 4} {
		t.Fatal("catalog search overwrote library state")
	}
}
