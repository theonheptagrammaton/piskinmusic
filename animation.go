package main

import "time"

func (m model) reducedMotion() bool {
	return configBool(m.cfg, "ui.reduced_motion", false)
}

func (m model) visualizerVisible() bool {
	return !m.blurred && m.phase == phaseReady && m.w >= 40 && m.h >= 12 &&
		!m.queueOpen && !m.eqOpen &&
		!m.helpOpen && !m.acctOpen && !m.searchOpen
}

func (m model) tickInterval() time.Duration {
	if m.blurred {
		return 250 * time.Millisecond
	}
	if m.st.Playing && m.visualizerVisible() &&
		(m.captureStatus() == captureLive || m.captureStatus() == captureDemo) {
		if m.reducedMotion() {
			return 100 * time.Millisecond
		}
		return time.Second / 30
	}
	if !m.reducedMotion() && (m.phase == phaseBoot || m.loading != "" || m.audioInitializing()) {
		return 100 * time.Millisecond
	}
	return 250 * time.Millisecond
}

// Use timer timestamps rather than counting frames: adaptive cadence must not
// slow progress, notification expiry, the initialization watchdog or Last.fm.
// A long suspend expires notices but earns at most one second of listening.
func (m *model) tickElapsed(now time.Time) time.Duration {
	elapsed := time.Second / 30
	if !now.IsZero() {
		if !m.lastTick.IsZero() {
			elapsed = max(0, now.Sub(m.lastTick))
		}
		if m.lastTick.IsZero() || now.After(m.lastTick) {
			m.lastTick = now
		}
	}
	m.t += elapsed.Seconds()
	return min(elapsed, time.Second)
}
