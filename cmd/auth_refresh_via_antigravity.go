package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ibrahim-wael/agswitch/internal/account"
	"github.com/ibrahim-wael/agswitch/internal/quota"
	"github.com/ibrahim-wael/agswitch/internal/switcher"
	"github.com/spf13/cobra"
)

type antigravityRefreshResult struct {
	Profile   string `json:"profile"`
	Email     string `json:"email,omitempty"`
	Status    string `json:"status"`
	Models    int    `json:"models,omitempty"`
	Remaining int    `json:"minimum_remaining,omitempty"`
	Source    string `json:"source,omitempty"`
	Error     string `json:"error,omitempty"`
}

type antigravityRefreshReport struct {
	OriginalProfile string                     `json:"original_profile"`
	Restored        bool                       `json:"restored"`
	Results         []antigravityRefreshResult `json:"results"`
}

func newAuthRefreshViaAntigravityCommand(d *dependencies) *cobra.Command {
	var all bool
	var confirmIdle bool
	var jsonOutput bool
	var timeout time.Duration
	command := &cobra.Command{
		Use:   "refresh-via-antigravity [profile]",
		Short: "Let Antigravity renew saved credentials, then refresh quota",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			if !confirmIdle {
				return fmt.Errorf("refresh-via-antigravity restarts the language server for each target; pass --confirm-idle only after the current response and tool calls have finished")
			}
			if all && len(args) != 0 {
				return fmt.Errorf("use either --all or one profile, not both")
			}
			if !all && len(args) == 0 {
				return fmt.Errorf("provide a profile or pass --all")
			}
			if timeout <= 0 {
				timeout = 25 * time.Second
			}

			accounts, err := d.app.List(command.Context())
			if err != nil {
				return err
			}
			current, found, err := d.app.Current(command.Context())
			if err != nil {
				return err
			}
			if !found {
				return fmt.Errorf("the active credential does not match a saved profile; cannot guarantee restoration")
			}

			targets, err := selectAntigravityRefreshTargets(accounts, args, all)
			if err != nil {
				return err
			}
			report, refreshErr := refreshProfilesViaAntigravity(command, d, current.ID, targets, timeout, jsonOutput)
			if jsonOutput {
				encoder := json.NewEncoder(command.OutOrStdout())
				encoder.SetIndent("", "  ")
				if err := encoder.Encode(report); err != nil {
					return errors.Join(refreshErr, err)
				}
			}
			return refreshErr
		},
	}
	command.Flags().BoolVar(&all, "all", false, "refresh every quota-enabled saved profile that is not already live")
	command.Flags().BoolVar(&confirmIdle, "confirm-idle", false, "confirm the current response and tool calls have finished")
	command.Flags().BoolVar(&jsonOutput, "json", false, "print JSON results")
	command.Flags().DurationVar(&timeout, "timeout", 25*time.Second, "maximum time to wait for Antigravity to renew each profile")
	return command
}

func selectAntigravityRefreshTargets(accounts []account.Account, args []string, all bool) ([]account.Account, error) {
	if all {
		out := make([]account.Account, 0, len(accounts))
		for _, item := range accounts {
			if item.QuotaEnabled {
				out = append(out, item)
			}
		}
		return out, nil
	}
	name := args[0]
	for _, item := range accounts {
		if item.ID == name {
			return []account.Account{item}, nil
		}
	}
	return nil, fmt.Errorf("profile not found: %s", name)
}

func refreshProfilesViaAntigravity(command *cobra.Command, d *dependencies, original string, targets []account.Account, timeout time.Duration, quiet bool) (report antigravityRefreshReport, err error) {
	report.OriginalProfile = original
	active := original

	defer func() {
		if active == original {
			report.Restored = true
			return
		}
		restoreErr := d.app.Use(command.Context(), original, switcher.Options{LaunchMode: switcher.PreserveLaunchState, HotReload: true})
		if restoreErr == nil {
			report.Restored = true
			active = original
		} else {
			err = errors.Join(err, fmt.Errorf("restore original profile %q: %w", original, restoreErr))
		}
	}()

	for _, target := range targets {
		initial := d.quota.Fetch(command.Context(), target.ID, true)
		if isRecentLiveQuota(initial) {
			result := summarizeAntigravityRefresh(target, "already_live", initial, "")
			report.Results = append(report.Results, result)
			if !quiet {
				fmt.Fprintf(command.OutOrStdout(), "[LIVE] %s already has recent quota (%d models)\n", target.ID, result.Models)
			}
			continue
		}

		if active != target.ID {
			if !quiet {
				fmt.Fprintf(command.OutOrStdout(), "[SWITCH] %s · restarting language server only\n", target.ID)
			}
			if switchErr := d.app.Use(command.Context(), target.ID, switcher.Options{LaunchMode: switcher.PreserveLaunchState, HotReload: true}); switchErr != nil {
				result := summarizeAntigravityRefresh(target, "switch_failed", quota.Result{Profile: target.ID}, switchErr.Error())
				report.Results = append(report.Results, result)
				err = errors.Join(err, fmt.Errorf("refresh %s: %w", target.ID, switchErr))
				continue
			}
			active = target.ID
		}

		refreshed, waitErr := waitForAntigravityCredentialRefresh(command, d, target, timeout)
		if waitErr != nil {
			result := summarizeAntigravityRefresh(target, "not_refreshed", refreshed, waitErr.Error())
			report.Results = append(report.Results, result)
			if !quiet {
				fmt.Fprintf(command.OutOrStdout(), "[AUTH] %s did not become live: %s\n", target.ID, waitErr)
			}
			continue
		}

		result := summarizeAntigravityRefresh(target, "live", refreshed, "")
		report.Results = append(report.Results, result)
		if !quiet {
			fmt.Fprintf(command.OutOrStdout(), "[LIVE] %s renewed via Antigravity · %d models", target.ID, result.Models)
			if result.Remaining >= 0 {
				fmt.Fprintf(command.OutOrStdout(), " · min %d%%", result.Remaining)
			}
			fmt.Fprintln(command.OutOrStdout())
		}
	}

	if active != original {
		if !quiet {
			fmt.Fprintf(command.OutOrStdout(), "[RESTORE] %s\n", original)
		}
		if restoreErr := d.app.Use(command.Context(), original, switcher.Options{LaunchMode: switcher.PreserveLaunchState, HotReload: true}); restoreErr != nil {
			return report, errors.Join(err, fmt.Errorf("restore original profile %q: %w", original, restoreErr))
		}
		active = original
		report.Restored = true
	}
	return report, err
}

func waitForAntigravityCredentialRefresh(command *cobra.Command, d *dependencies, target account.Account, timeout time.Duration) (quota.Result, error) {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(750 * time.Millisecond)
	defer ticker.Stop()
	originalFingerprint := target.CredentialFingerprint
	var last quota.Result
	var lastReason string

	for {
		items, err := d.app.List(command.Context())
		if err != nil {
			return last, err
		}
		for _, item := range items {
			if item.ID != target.ID {
				continue
			}
			credentialChanged := item.CredentialFingerprint != "" && item.CredentialFingerprint != originalFingerprint
			if credentialChanged {
				last = d.quota.Fetch(command.Context(), target.ID, true)
				if isRecentLiveQuota(last) {
					return last, nil
				}
				lastReason = quotaResultReason(last)
			}
			break
		}

		select {
		case <-command.Context().Done():
			return last, command.Context().Err()
		case <-deadline.C:
			if lastReason == "" {
				lastReason = "Antigravity did not publish a renewed active credential before the timeout"
			}
			return last, errors.New(lastReason)
		case <-ticker.C:
		}
	}
}

func isRecentLiveQuota(result quota.Result) bool {
	if result.Err != nil {
		return false
	}
	ok, _ := quota.EligibleForAutoSwitch(result.Snapshot, time.Now().UTC(), quota.DefaultAutoSwitchMaxAge)
	return ok
}

func quotaResultReason(result quota.Result) string {
	if result.Err != nil {
		return result.Err.Error()
	}
	if warning := strings.TrimSpace(result.Snapshot.Metadata["warning"]); warning != "" {
		return warning
	}
	if ok, reason := quota.EligibleForAutoSwitch(result.Snapshot, time.Now().UTC(), quota.DefaultAutoSwitchMaxAge); !ok {
		return reason
	}
	return "quota did not become live"
}

func summarizeAntigravityRefresh(item account.Account, status string, result quota.Result, errorText string) antigravityRefreshResult {
	out := antigravityRefreshResult{
		Profile:   item.ID,
		Email:     item.Email,
		Status:    status,
		Remaining: -1,
		Error:     errorText,
	}
	if result.Snapshot.Email != "" {
		out.Email = result.Snapshot.Email
	}
	out.Models = len(result.Snapshot.Models)
	out.Source = result.Snapshot.Source
	if remaining, ok := quota.MinimumKnownRemaining(result.Snapshot); ok {
		out.Remaining = remaining
	}
	return out
}
