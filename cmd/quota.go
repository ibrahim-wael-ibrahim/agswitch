package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ibrahim-wael/agswitch/internal/account"
	"github.com/ibrahim-wael/agswitch/internal/quota"
	"github.com/spf13/cobra"
)

type quotaJSONResult struct {
	Profile  string         `json:"profile"`
	Snapshot quota.Snapshot `json:"snapshot,omitempty"`
	Error    string         `json:"error,omitempty"`
}

func newQuotaCommand(dependencies *dependencies) *cobra.Command {
	var refresh bool
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "quota [profile]",
		Short: "Show live model quota for saved profiles",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			accounts, err := dependencies.app.List(command.Context())
			if err != nil {
				return err
			}
			if len(args) == 1 {
				accounts, err = selectQuotaAccount(accounts, args[0])
				if err != nil {
					return err
				}
			}
			results := dependencies.quota.FetchAll(command.Context(), accounts, refresh)
			if jsonOutput {
				output := make([]quotaJSONResult, len(results))
				for index, result := range results {
					output[index] = quotaJSONResult{Profile: result.Profile, Snapshot: result.Snapshot}
					if result.Err != nil {
						output[index].Error = result.Err.Error()
					}
				}
				return json.NewEncoder(command.OutOrStdout()).Encode(output)
			}
			failed := false
			for index, result := range results {
				if index > 0 {
					fmt.Fprintln(command.OutOrStdout())
				}
				if result.Err != nil {
					failed = true
					fmt.Fprintf(command.ErrOrStderr(), "\033[1;31m[FAIL]\033[0m %s: %v\n", result.Profile, result.Err)
					continue
				}
				printQuota(command, result.Snapshot)
			}
			if failed {
				return fmt.Errorf("one or more quota requests failed")
			}
			return nil
		},
	}
	command.Flags().BoolVarP(&refresh, "refresh", "r", false, "bypass the five-minute cache")
	command.Flags().BoolVar(&jsonOutput, "json", false, "print JSON")
	return command
}

func selectQuotaAccount(accounts []account.Account, profile string) ([]account.Account, error) {
	for _, item := range accounts {
		if item.ID == profile {
			return []account.Account{item}, nil
		}
	}
	return nil, fmt.Errorf("profile not found: %s", profile)
}

func printQuota(command *cobra.Command, snapshot quota.Snapshot) {
	email := strings.TrimSpace(snapshot.Email)
	if email == "" {
		email = snapshot.Profile
	}
	fmt.Fprintf(command.OutOrStdout(), "\033[1;36m%s\033[0m  \033[2m[%s]\033[0m\n", email, snapshot.Profile)
	if snapshot.SubscriptionTier != "" {
		fmt.Fprintf(command.OutOrStdout(), "Plan: %s\n", snapshot.SubscriptionTier)
	}
	fmt.Fprintf(command.OutOrStdout(), "Source: %s  Updated: %s\n\n", snapshot.Source, snapshot.FetchedAt.Local().Format("2006-01-02 15:04:05"))
	for _, model := range quota.SortedModels(snapshot) {
		remaining := "--"
		if model.Remaining >= 0 {
			remaining = fmt.Sprintf("%d%%", model.Remaining)
		}
		reset := ""
		if duration := quota.ResetIn(model.ResetAt, time.Now()); duration > 0 {
			reset = "reset in " + duration.Round(time.Minute).String()
		}
		fmt.Fprintf(command.OutOrStdout(), "%-34s %s  %s\n", model.Name, remaining, reset)
	}
}
