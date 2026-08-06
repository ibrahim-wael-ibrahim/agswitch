package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

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
	AutoRefresh     time.Duration
	ExitAfterSwitch bool
}

type focusArea int

const (
	focusCommands focusArea = iota
	focusAccounts
	focusSearch
	focusRefreshInput
	focusThresholdInput
	focusConfirm
)

type action string

const (
	actionHotReload     action = "hot-reload"
	actionSwitchLaunch  action = "switch-launch"
	actionSwitchOnly    action = "switch-only"
	actionUpdate        action = "update"
	actionRefresh       action = "refresh"
	actionAutoRefresh   action = "auto-refresh"
	actionAutoSwitch    action = "auto-switch"
	actionAutoThreshold action = "auto-threshold"
	actionPrevious      action = "previous"
	actionDoctor        action = "doctor"
	actionQuit          action = "quit"
)

type commandItem struct {
	Action      action
	Icon        string
	Label       string
	Description string
	Shortcut    string
}

var dashboardCommands = []commandItem{
	{Action: actionHotReload, Icon: "⚡", Label: "Hot switch", Description: "Keep the Antigravity UI, files, chat and terminals open", Shortcut: "s"},
	{Action: actionSwitchLaunch, Icon: "↻", Label: "Full restart", Description: "Restart Antigravity and clean old language servers"},
	{Action: actionSwitchOnly, Icon: "○", Label: "Activate only", Description: "Change the keyring credential without starting Antigravity"},
	{Action: actionUpdate, Icon: "↓", Label: "Sync profile", Description: "Save Antigravity's renewed active credential into this profile"},
	{Action: actionRefresh, Icon: "⟳", Label: "Refresh quota", Description: "Fetch live model quota now", Shortcut: "r"},
	{Action: actionAutoSwitch, Icon: "★", Label: "Auto hot-switch", Description: "Use the safest recent live-quota recommendation", Shortcut: "a"},
	{Action: actionAutoThreshold, Icon: "%", Label: "Auto threshold", Description: "Choose when auto-switch becomes eligible", Shortcut: "A"},
	{Action: actionAutoRefresh, Icon: "◷", Label: "Refresh timer", Description: "Set automatic live-quota refresh seconds", Shortcut: "R"},
	{Action: actionPrevious, Icon: "←", Label: "Previous account", Description: "Return to the previously active profile", Shortcut: "p"},
	{Action: actionDoctor, Icon: "✓", Label: "Run doctor", Description: "Check keyring, paths and process state", Shortcut: "d"},
	{Action: actionQuit, Icon: "×", Label: "Quit", Description: "Close AGSwitch without changing accounts", Shortcut: "q"},
}

type Model struct {
	Context context.Context
	Backend Backend
	Options Options

	Accounts []account.Account
	Results  []quota.Result
	Decision autoswitch.Decision

	SelectedAccount  int
	SelectedCommand  int
	Focus            focusArea
	Search           string
	Searching        bool
	RefreshInput     string
	EditingRefresh   bool
	ThresholdInput   string
	EditingThreshold bool
	ConfirmAction    action
	ConfirmProfile   string
	ConfirmTitle     string
	ConfirmBody      string
	Status           string
	Details          string
	Width            int
	Height           int
	Busy             bool
	Initialized      bool
	RefreshSequence  uint64
}

func New(ctx context.Context, backend Backend, options Options) Model {
	if options.AutoThreshold < 0 {
		options.AutoThreshold = 0
	}
	if options.AutoThreshold > 100 {
		options.AutoThreshold = 100
	}
	if options.AutoRefresh < 0 {
		options.AutoRefresh = 0
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

type autoRefreshMsg struct {
	sequence uint64
}

func (m Model) scheduleAutoRefresh() tea.Cmd {
	if m.Options.AutoRefresh <= 0 {
		return nil
	}
	sequence := m.RefreshSequence
	return tea.Tick(m.Options.AutoRefresh, func(time.Time) tea.Msg {
		return autoRefreshMsg{sequence: sequence}
	})
}

func (m Model) setAutoRefreshSeconds(value string) error {
	seconds, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || seconds < 0 {
		return fmt.Errorf("enter a whole number of seconds, or 0 to disable")
	}
	m.Options.AutoRefresh = time.Duration(seconds) * time.Second
	return nil
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
	if m.ConfirmProfile != "" {
		profile = m.ConfirmProfile
		hasProfile = true
	}
	return func() tea.Msg {
		result := operationMsg{action: selected, profile: profile}
		switch selected {
		case actionHotReload:
			if !hasProfile {
				result.err = fmt.Errorf("select an account first")
				return result
			}
			result.err = m.Backend.Use(m.Context, profile, switcher.Options{
				LaunchMode: switcher.PreserveLaunchState,
				HotReload:  true,
			})
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
				result.profile = ""
				return result
			}
			result.profile = m.Decision.Selected.Profile
			result.err = m.Backend.Use(m.Context, result.profile, switcher.Options{
				LaunchMode: switcher.PreserveLaunchState,
				HotReload:  true,
			})
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

func (m Model) activeProfile() string {
	for _, item := range m.Accounts {
		if item.Active {
			return item.ID
		}
	}
	return ""
}

func (m Model) confirming() bool {
	return m.ConfirmAction != ""
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
