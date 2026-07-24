package tui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/ibrahim-wael/agswitch/internal/account"
	"github.com/ibrahim-wael/agswitch/internal/autoswitch"
	"github.com/ibrahim-wael/agswitch/internal/doctor"
	"github.com/ibrahim-wael/agswitch/internal/quota"
	"github.com/ibrahim-wael/agswitch/internal/switcher"
)

type Backend interface {
	List(context.Context) ([]account.Account, error)
	Use(context.Context, string, switcher.Options) error
	Update(context.Context, string) error
	FetchAll(context.Context, []account.Account, bool) []quota.Result
	Previous(context.Context) (string, error)
	Doctor(context.Context) []doctor.Check
}

type Options struct {
	Version         string
	AutoThreshold   int
	ExitAfterSwitch bool
}

type focusArea int

const (
	focusCommands focusArea = iota
	focusAccounts
	focusSearch
)

type action string

const (
	actionSwitchLaunch action = "switch-launch"
	actionSwitchOnly   action = "switch-only"
	actionUpdate       action = "update"
	actionRefresh      action = "refresh"
	actionAutoSwitch   action = "auto-switch"
	actionPrevious     action = "previous"
	actionDoctor       action = "doctor"
	actionQuit         action = "quit"
)

type commandItem struct {
	Action      action
	Label       string
	Description string
}

var dashboardCommands = []commandItem{
	{Action: actionSwitchLaunch, Label: "Switch + launch", Description: "Activate the selected account and start Antigravity"},
	{Action: actionSwitchOnly, Label: "Switch only", Description: "Activate the selected account without launching"},
	{Action: actionUpdate, Label: "Update profile", Description: "Save the current Antigravity credential into this profile"},
	{Action: actionRefresh, Label: "Refresh quota", Description: "Bypass cache and fetch live model quota"},
	{Action: actionAutoSwitch, Label: "Auto switch", Description: "Apply the conservative quota recommendation"},
	{Action: actionPrevious, Label: "Previous account", Description: "Return to the previously active profile"},
	{Action: actionDoctor, Label: "Run doctor", Description: "Check platform, keyring, paths and application state"},
	{Action: actionQuit, Label: "Quit dashboard", Description: "Close agswitch without changing the account"},
}

type Model struct {
	Context context.Context
	Backend Backend
	Options Options

	Accounts []account.Account
	Results  []quota.Result
	Decision autoswitch.Decision

	SelectedAccount int
	SelectedCommand int
	Focus           focusArea
	Search          string
	Searching       bool
	Status          string
	Details         string
	Width           int
	Height          int
	Busy            bool
}

func New(ctx context.Context, backend Backend, options Options) Model {
	if options.AutoThreshold < 0 {
		options.AutoThreshold = 0
	}
	if options.AutoThreshold > 100 {
		options.AutoThreshold = 100
	}
	return Model{
		Context: ctx,
		Backend: backend,
		Options: options,
		Focus:   focusAccounts,
		Status:  "Loading accounts and quota...",
		Busy:    true,
	}
}

func Run(ctx context.Context, backend Backend, options Options) error {
	if backend == nil {
		return fmt.Errorf("dashboard backend is not configured")
	}
	program := tea.NewProgram(New(ctx, backend, options))
	_, err := program.Run()
	return err
}

func (m Model) Init() tea.Cmd {
	return m.loadDataCommand(false)
}

type dataMsg struct {
	accounts []account.Account
	results  []quota.Result
	decision autoswitch.Decision
	err      error
	refresh  bool
}

type operationMsg struct {
	action  action
	profile string
	details string
	err     error
}

func (m Model) loadDataCommand(refresh bool) tea.Cmd {
	return func() tea.Msg {
		accounts, err := m.Backend.List(m.Context)
		if err != nil {
			return dataMsg{err: err, refresh: refresh}
		}
		results := m.Backend.FetchAll(m.Context, accounts, refresh)
		current := ""
		for _, item := range accounts {
			if item.Active {
				current = item.ID
				break
			}
		}
		return dataMsg{
			accounts: accounts,
			results:  results,
			decision: autoswitch.Select(results, current, m.Options.AutoThreshold),
			refresh:  refresh,
		}
	}
}

func (m Model) operationCommand(selected action) tea.Cmd {
	profile, hasProfile := m.selectedProfile()
	return func() tea.Msg {
		result := operationMsg{action: selected, profile: profile}
		switch selected {
		case actionSwitchLaunch:
			if !hasProfile {
				result.err = fmt.Errorf("select an account first")
				return result
			}
			result.err = m.Backend.Use(m.Context, profile, switcher.Options{LaunchMode: switcher.AlwaysLaunch})
		case actionSwitchOnly:
			if !hasProfile {
				result.err = fmt.Errorf("select an account first")
				return result
			}
			result.err = m.Backend.Use(m.Context, profile, switcher.Options{LaunchMode: switcher.NeverLaunch})
		case actionUpdate:
			if !hasProfile {
				result.err = fmt.Errorf("select an account first")
				return result
			}
			result.err = m.Backend.Update(m.Context, profile)
		case actionAutoSwitch:
			if !m.Decision.Switch || strings.TrimSpace(m.Decision.Selected.Profile) == "" {
				result.details = m.Decision.Reason
				return result
			}
			result.profile = m.Decision.Selected.Profile
			result.err = m.Backend.Use(m.Context, result.profile, switcher.Options{LaunchMode: switcher.AlwaysLaunch})
		case actionPrevious:
			result.profile, result.err = m.Backend.Previous(m.Context)
		case actionDoctor:
			checks := m.Backend.Doctor(m.Context)
			okCount, warnCount, failCount := 0, 0, 0
			lines := make([]string, 0, len(checks))
			for _, check := range checks {
				switch check.Status {
				case doctor.OK:
					okCount++
				case doctor.Warn:
					warnCount++
				case doctor.Fail:
					failCount++
				}
				lines = append(lines, fmt.Sprintf("[%s] %s: %s", check.Status, check.Name, check.Details))
			}
			result.details = fmt.Sprintf("Doctor: %d OK, %d warnings, %d failures\n%s", okCount, warnCount, failCount, strings.Join(lines, "\n"))
		}
		return result
	}
}

func (m Model) selectedProfile() (string, bool) {
	filtered := m.filteredAccounts()
	if len(filtered) == 0 {
		return "", false
	}
	index := m.SelectedAccount
	if index < 0 || index >= len(filtered) {
		index = 0
	}
	return filtered[index].ID, true
}

func (m Model) filteredAccounts() []account.Account {
	query := strings.ToLower(strings.TrimSpace(m.Search))
	if query == "" {
		return append([]account.Account(nil), m.Accounts...)
	}
	items := make([]account.Account, 0, len(m.Accounts))
	for _, item := range m.Accounts {
		haystack := strings.ToLower(item.ID + " " + item.Email + " " + item.Label)
		if strings.Contains(haystack, query) {
			items = append(items, item)
		}
	}
	return items
}

func (m Model) resultFor(profile string) quota.Result {
	for _, result := range m.Results {
		if result.Profile == profile {
			return result
		}
	}
	return quota.Result{Profile: profile, Err: fmt.Errorf("quota has not been loaded")}
}
