package tui

import "github.com/ibrahim-wael/agswitch/internal/account"

func (m *Model) SetAccounts(a []account.Account) {
	m.Accounts = append([]account.Account(nil), a...)
	if m.Selected >= len(m.Accounts) {
		m.Selected = 0
	}
}
func (m Model) SelectedAccount() (account.Account, bool) {
	if len(m.Accounts) == 0 {
		return account.Account{}, false
	}
	i := m.Selected
	if i < 0 || i >= len(m.Accounts) {
		i = 0
	}
	return m.Accounts[i], true
}
