package main

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) updateMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.phase != phaseReady || m.helpOpen || m.acctOpen || m.searchOpen {
		return m, nil
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
		switch {
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
