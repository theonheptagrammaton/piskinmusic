package main

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) updateMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.phase != phaseReady || msg.X < 0 || msg.Y < 0 || msg.X >= m.w || msg.Y >= m.h {
		return m, nil
	}
	if m.helpOpen {
		return m.helpMouse(msg)
	}
	if m.acctOpen {
		if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
			// Account's explicit sign-out row is shared with its renderer.
			if m.accountAction().contains(msg.X, msg.Y) {
				return m.updateAccount(keyMessage("enter"))
			}
			m.acctOpen = false
		}
		return m, nil
	}
	if m.searchOpen {
		return m.searchMouse(msg)
	}
	lay := m.layout()
	if !lay.visible {
		return m, nil
	}
	inQueue := lay.queueBody.contains(msg.X, msg.Y)
	inRecent := lay.recentBody.contains(msg.X, msg.Y)

	switch msg.Button {
	case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}
		step := 1
		if msg.Button == tea.MouseButtonWheelUp {
			step = -1
		}
		if m.eqOpen {
			if !lay.main.contains(msg.X, msg.Y) {
				return m, nil
			}
			key := "up"
			if step > 0 {
				key = "down"
			}
			return m.updateKeys(keyMessage(key))
		}
		switch {
		case lay.libraryBody.contains(msg.X, msg.Y):
			m.focus = focusLibrary
			m.librarySel[m.libraryTab] = max(0, min(m.librarySel[m.libraryTab]+step, len(m.libraryList(m.libraryTab))-1))
		case lay.transport.contains(msg.X, msg.Y):
			m.focus = focusPlayer
			key := "up"
			if step > 0 {
				key = "down"
			}
			return m.updateKeys(keyMessage(key))
		case inQueue:
			m.focus = focusQueue
			m.selIdx = max(0, min(m.selIdx+step, len(m.st.Queue)-1))
		case inRecent:
			m.focus = focusRecent
			m.recentSel = max(0, min(m.recentSel+step*recentCols, len(m.recent)-1))
		}
		return m, nil

	case tea.MouseButtonLeft:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}
		if m.eqOpen {
			return m.eqMouse(msg, lay)
		}
		for i, r := range lay.nav {
			if r.contains(msg.X, msg.Y) && !m.queueOpen {
				m.selectScreen(screen(i))
				return m, nil
			}
		}
		for _, a := range lay.actions {
			if a.contains(msg.X, msg.Y) {
				return m.updateKeys(keyMessage(a.key))
			}
		}
		for i, r := range lay.libraryTabs {
			if r.contains(msg.X, msg.Y) {
				m.libraryTab, m.focus = i, focusLibrary
				return m, nil
			}
		}
		if lay.libraryBody.contains(msg.X, msg.Y) {
			list := m.libraryList(m.libraryTab)
			i := listWindow(m.librarySel[m.libraryTab], len(list), lay.libraryBody.h) + msg.Y - lay.libraryBody.y
			if i < len(list) {
				m.librarySel[m.libraryTab], m.focus = i, focusLibrary
				return m.playLibrarySelection()
			}
			return m, nil
		}
		if lay.play.contains(msg.X, msg.Y) {
			return m.updateKeys(keyMessage("space"))
		}
		for _, a := range m.transportActions(lay) {
			if a.contains(msg.X, msg.Y) {
				return m.updateKeys(keyMessage(a.key))
			}
		}

		return m.click(msg, lay, inQueue, inRecent)
	}
	return m, nil
}

func (m model) click(msg tea.MouseMsg, lay layout, inQueue, inRecent bool) (tea.Model, tea.Cmd) {
	eng := m.eng
	switch {
	case inQueue && len(m.st.Queue) > 0:
		// Same windowing queuePanel uses, so the row under the pointer is the
		// track that was drawn there.
		rows := lay.queueRows()
		start := max(0, min(m.selIdx-rows/2, len(m.st.Queue)-rows))
		i := start + msg.Y - lay.queueTop()
		if i < 0 || i >= len(m.st.Queue) {
			return m, nil
		}
		m.focus, m.selIdx = focusQueue, i
		if m.audioInitializing() {
			m.note, m.noteAt = audioInitializingText, m.t
			return m, nil
		}
		m.loading = m.st.Queue[i].Title
		m.loadSnap, m.loadStart, m.loadHideQueue = snap(m.st), m.t, false
		return m, doCmd(func() error { return eng.JumpTo(i) })

	case inRecent:
		i := lay.recentTileAt(msg.X, msg.Y, m.recentSel, len(m.recent))
		if i < 0 {
			return m, nil
		}
		m.focus, m.recentSel = focusRecent, i
		if m.audioInitializing() {
			m.note, m.noteAt = audioInitializingText, m.t
			return m, nil
		}
		tr := m.recent[i]
		m.loading, m.loadSnap, m.loadStart = tr.Title, snap(m.st), m.t
		m.loadHideQueue = true
		m.beginAudioInitialization()
		kind, id := tr.Kind, tr.ID
		return m, doCmd(func() error { return eng.Play(kind, id) })

	case lay.progress.contains(msg.X, msg.Y) && m.st.Dur > 0:
		// The transport's second line. Seeking is the one thing a pointer does
		// better than a keyboard, so it is worth the geometry.
		x, barW := msg.X-lay.progress.x, lay.progress.w
		m.focus = focusPlayer
		m.st.Pos = time.Duration(float64(m.st.Dur) * float64(x) / float64(barW))
		m.wv.reset() // the recorded waveform no longer matches the new position
		pos := m.st.Pos
		return m, doCmd(func() error { return eng.SeekTo(pos) })
	}
	return m, nil
}

// keyMessage keeps clickable actions on the same command path as the keyboard.
func keyMessage(key string) tea.KeyMsg {
	switch key {
	case "space":
		return tea.KeyMsg{Type: tea.KeySpace}
	case "enter", "↵":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "S-tab", "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "left", "←":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right", "→":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "up", "↑":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down", "↓":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
}
func (m model) helpMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Action != tea.MouseActionPress {
		return m, nil
	}
	if msg.Button == tea.MouseButtonWheelDown {
		return m.updateHelp(keyMessage("down"))
	}
	if msg.Button == tea.MouseButtonWheelUp {
		return m.updateHelp(keyMessage("up"))
	}
	if msg.Button != tea.MouseButtonLeft {
		return m, nil
	}
	for _, a := range m.helpTargets() {
		if a.contains(msg.X, msg.Y) {
			m.helpOpen = false
			key := a.key
			if strings.HasPrefix(key, "player:") {
				m.focus = focusPlayer
				m.eqOpen = false
				key = strings.TrimPrefix(key, "player:")
			}
			if strings.HasPrefix(key, "eq:") {
				m.eqOpen = true
				key = strings.TrimPrefix(key, "eq:")
			}
			return m.updateKeys(keyMessage(key))
		}
	}
	if msg.Y == m.h-2 {
		if msg.X < 8 {
			return m.updateHelp(keyMessage("left"))
		}
		if msg.X < 29 {
			return m.updateHelp(keyMessage("right"))
		}
	}
	m.helpOpen = false
	return m, nil
}
func (m model) searchMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Action != tea.MouseActionPress {
		return m, nil
	}
	body, tabs, actions := m.searchGeometry()
	list := m.curList()
	if msg.Button == tea.MouseButtonWheelDown || msg.Button == tea.MouseButtonWheelUp {
		if !body.contains(msg.X, msg.Y) {
			return m, nil
		}
		step := 1
		if msg.Button == tea.MouseButtonWheelUp {
			step = -1
		}
		m.sInput = false
		m.sSel = max(0, min(m.sSel+step, len(list)-1))
		return m, nil
	}
	if msg.Button != tea.MouseButtonLeft {
		return m, nil
	}
	for i, t := range tabs {
		if t.contains(msg.X, msg.Y) {
			m.sTab, m.sSel = i, 0
			return m, nil
		}
	}
	for _, a := range actions {
		if a.contains(msg.X, msg.Y) {
			if a.key == "a" || a.key == "A" {
				m.sInput = false
			}
			return m.updateSearch(keyMessage(a.key))
		}
	}
	if msg.Y == 1 {
		m.sInput = true
		return m, nil
	}
	if body.contains(msg.X, msg.Y) && !m.sBusy {
		i := listWindow(m.sSel, len(list), body.h) + msg.Y - body.y
		if i < len(list) {
			m.sSel, m.sInput = i, false
			return m.updateSearch(keyMessage("enter"))
		}
	}
	return m, nil
}
func (m model) eqMouse(msg tea.MouseMsg, l layout) (tea.Model, tea.Cmd) {
	if l.play.contains(msg.X, msg.Y) {
		return m.updateKeys(keyMessage("space"))
	}
	if l.progress.contains(msg.X, msg.Y) {
		return m.click(msg, l, false, false)
	}
	for _, a := range m.transportActions(l) {
		if a.contains(msg.X, msg.Y) {
			return m.updateKeys(keyMessage(a.key))
		}
	}

	for _, a := range l.actions {
		if a.contains(msg.X, msg.Y) {
			return m.updateKeys(keyMessage(a.key))
		}
	}
	for _, a := range m.eqActions(l) {
		if a.contains(msg.X, msg.Y) {
			return m.updateKeys(keyMessage(a.key))
		}
	}
	if (rect{l.main.x + 1, l.main.y + 2, l.main.w - 2, l.main.h - 4}).contains(msg.X, msg.Y) {
		colW := max(2, (l.main.w-3)/10-1) + 1
		m.eqSel = min(9, max(0, (msg.X-l.main.x-1)/colW))
	}
	return m, nil
}

// Help key tokens are hit-tested separately: clicking p in "n / p" runs p.
func helpKeyTokens(keys string) []string { return strings.Fields(strings.ReplaceAll(keys, "/", " / ")) }
