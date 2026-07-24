package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/ibrahim-wael/agswitch/internal/account"
	"github.com/ibrahim-wael/agswitch/internal/quota"
	"github.com/ibrahim-wael/agswitch/internal/switcher"
)

type Backend interface {
	List(context.Context) ([]account.Account, error)
	Use(context.Context, string, switcher.Options) error
}

type Model struct {
	Context   context.Context
	Backend   Backend
	Accounts  []account.Account
	Selected  int
	Status    string
	Width     int
	Height    int
	Stay      bool
	Switching bool
	Snapshot  quota.Snapshot
}

func New(ctx context.Context, backend Backend, accounts []account.Account, stay bool) Model {
	return Model{
		Context:  ctx,
		Backend:  backend,
		Accounts: append([]account.Account(nil), accounts...),
		Status:   "Ready",
		Stay:     stay,
	}
}

func Run(ctx context.Context, backend Backend, stay bool) error {
	accounts, err := backend.List(ctx)
	if err != nil {
		return err
	}

	program := tea.NewProgram(New(ctx, backend, accounts, stay))
	_, err = program.Run()
	return err
}

func (m Model) Init() tea.Cmd {
	return nil
}

type switchResultMsg struct {
	profile string
	err     error
}

func (m Model) switchCommand(profile string) tea.Cmd {
	return func() tea.Msg {
		return switchResultMsg{
			profile: profile,
			err: m.Backend.Use(m.Context, profile, switcher.Options{
				LaunchMode: switcher.AlwaysLaunch,
			}),
		}
	}
}
