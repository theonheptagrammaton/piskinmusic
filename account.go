package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// updateAccount runs while the account overlay is open. Unlike help, this
// overlay has an action in it, so a stray key must not fire it — only enter
// signs out, and everything else closes.
func (m model) updateAccount(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "enter" && m.eng != nil && !m.acctBusy {
		m.acctBusy = true
		return m, signOutCmd(m.eng)
	}
	m.acctOpen = false
	return m, nil
}

// accountLines is the body of the overlay: who is signed in, then the action.
// Fetching the name is a round trip to Apple, so the overlay opens immediately
// and says so rather than waiting for it.
func (m model) accountLines() []string {
	who := "signed in"
	switch {
	case m.acctBusy:
		who = "signing out…"
	case m.acct.Name != "" && m.acct.Handle != "":
		who = m.acct.Name + " · @" + m.acct.Handle
	case m.acct.Name != "":
		who = m.acct.Name
	case m.acctLoading:
		who = "reading account…"
	}
	where := ""
	if m.acct.Storefront != "" {
		where = m.acct.Storefront
		if m.acct.Country != "" {
			where += " (" + m.acct.Country + ")"
		}
	} else if m.acct.Country != "" {
		where = m.acct.Country
	}
	return []string{who, where}
}

func (m model) accountAction() rect { return (rect{1, 5, 10, 1}).clipped(m.w, m.h) }
func (m model) accountView() string {
	lines := m.accountLines()
	action := " ↵ Sign out"
	if m.acctBusy {
		action = " Signing out…"
	}
	body := []string{"PISKIN MUSIC · ACCOUNT", "", lines[0], lines[1], "", action, "", "Enter signs out · other key closes"}
	return strings.Join(body, "\n")
}
