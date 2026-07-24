package tui

import "github.com/ibrahim-wael/agswitch/internal/account"

func (m *Model) SetAccounts(accounts []account.Account) {
	m.Accounts = append([]account.Account(nil), accounts...)
	if m.Selected >= len(m.Accounts) {
		m.Selected = 0
	}
}

func (m Model) SelectedAccount() (account.Account, bool) {
	if len(m.Accounts) == 0 {
		return account.Account{}, false
	}

	selected := m.Selected
	if selected < 0 || selected >= len(m.Accounts) {
		selected = 0
	}

	return m.Accounts[selected], true
}
