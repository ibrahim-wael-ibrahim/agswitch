package tui

import (
	"context"
	"testing"

	"github.com/ibrahim-wael/agswitch/internal/account"
	"github.com/ibrahim-wael/agswitch/internal/autoswitch"
	"github.com/ibrahim-wael/agswitch/internal/doctor"
	"github.com/ibrahim-wael/agswitch/internal/quota"
	"github.com/ibrahim-wael/agswitch/internal/switcher"
)

type hotReloadBackend struct {
	profile string
	options switcher.Options
}

func (b *hotReloadBackend) List(context.Context) ([]account.Account, error) { return nil, nil }
func (b *hotReloadBackend) Use(_ context.Context, profile string, options switcher.Options) error {
	b.profile = profile
	b.options = options
	return nil
}
func (b *hotReloadBackend) Update(context.Context, string) error { return nil }
func (b *hotReloadBackend) FetchAll(context.Context, []account.Account, bool) []quota.Result {
	return nil
}
func (b *hotReloadBackend) Previous(context.Context) (string, error) { return "", nil }
func (b *hotReloadBackend) Doctor(context.Context) []doctor.Check { return nil }

func TestHotSwitchIsPrimaryDashboardAction(t *testing.T) {
	if len(dashboardCommands) == 0 || dashboardCommands[0].Action != actionHotReload {
		t.Fatalf("first action = %#v; want hot reload", dashboardCommands)
	}
	if dashboardCommands[0].Shortcut != "s" {
		t.Fatalf("hot reload shortcut = %q; want s", dashboardCommands[0].Shortcut)
	}
}

func TestHotReloadOperationUsesBackendReload(t *testing.T) {
	backend := &hotReloadBackend{}
	model := Model{
		Context:        context.Background(),
		Backend:        backend,
		Accounts:       []account.Account{{ID: "work"}},
		ConfirmProfile: "work",
	}
	message := model.operationCommand(actionHotReload)()
	result, ok := message.(operationMsg)
	if !ok {
		t.Fatalf("operation returned %T", message)
	}
	if result.err != nil {
		t.Fatal(result.err)
	}
	if backend.profile != "work" || !backend.options.HotReload {
		t.Fatalf("profile=%q options=%#v", backend.profile, backend.options)
	}
	if backend.options.LaunchMode != switcher.PreserveLaunchState {
		t.Fatalf("launch mode=%v; want preserve", backend.options.LaunchMode)
	}
}

func TestHotSwitchRequiresConfirmation(t *testing.T) {
	model := Model{
		Accounts: []account.Account{{ID: "work"}},
	}
	updated, command := model.beginConfirmation(actionHotReload, "work")
	if command != nil {
		t.Fatal("confirmation must not execute immediately")
	}
	updatedModel := updated.(Model)
	if updatedModel.ConfirmAction != actionHotReload || updatedModel.ConfirmProfile != "work" {
		t.Fatalf("confirmation state = %#v", updatedModel)
	}
	if updatedModel.Focus != focusConfirm {
		t.Fatalf("focus = %v; want confirmation", updatedModel.Focus)
	}
}

func TestAutoSwitchUsesHotReloadAfterConfirmation(t *testing.T) {
	backend := &hotReloadBackend{}
	model := Model{
		Context:        context.Background(),
		Backend:        backend,
		Decision:       autoswitch.Decision{Switch: true, Selected: autoswitch.Candidate{Profile: "work"}},
		ConfirmProfile: "work",
	}
	message := model.operationCommand(actionAutoSwitch)()
	result := message.(operationMsg)
	if result.err != nil {
		t.Fatal(result.err)
	}
	if !backend.options.HotReload || backend.profile != "work" {
		t.Fatalf("profile=%q options=%#v", backend.profile, backend.options)
	}
}
