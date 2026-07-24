package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/ibrahim-wael/agswitch/internal/account"
	"github.com/ibrahim-wael/agswitch/internal/quota"
)

type Model struct {
	Accounts []account.Account
	Selected int
	Status   string
	Width    int
	Height   int
	Snapshot quota.Snapshot
}

func New() Model {
	return Model{Status: "Ready"}
}

func Run() error {
	program := tea.NewProgram(New())
	_, err := program.Run()
	return err
}

func (m Model) Init() tea.Cmd {
	return nil
}
