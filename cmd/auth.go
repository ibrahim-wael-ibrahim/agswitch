package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/ibrahim-wael/agswitch/internal/quota"
	"github.com/spf13/cobra"
)

func newAuthCommand(dependencies *dependencies) *cobra.Command {
	command := &cobra.Command{
		Use:   "auth",
		Short: "Inspect and renew saved OAuth authentication safely",
		Args:  cobra.NoArgs,
	}
	command.AddCommand(
		newAuthDoctorCommand(dependencies),
		newAuthRefreshViaAntigravityCommand(dependencies),
	)
	return command
}

func newAuthDoctorCommand(dependencies *dependencies) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose OAuth refresh, token client and quota access for every saved profile",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			accounts, err := dependencies.app.List(command.Context())
			if err != nil {
				return err
			}
			results := dependencies.quota.DiagnoseAllAuth(command.Context(), accounts)
			if jsonOutput {
				encoder := json.NewEncoder(command.OutOrStdout())
				encoder.SetIndent("", "  ")
				return encoder.Encode(results)
			}
			printAuthDiagnostics(command, results)
			return nil
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "print sanitized JSON diagnostics")
	return command
}

func printAuthDiagnostics(command *cobra.Command, results []quota.AuthDiagnostic) {
	live := 0
	refreshOK := 0
	failed := 0
	for _, result := range results {
		if result.QuotaStatus == "ok" {
			live++
		}
		if result.RefreshStatus == "ok" {
			refreshOK++
		}
		if result.RefreshStatus == "failed" || result.Error != "" {
			failed++
		}
	}
	fmt.Fprintf(command.OutOrStdout(), "OAuth diagnostics: %d profiles · %d quota-ready · %d refresh OK · %d failed\n", len(results), live, refreshOK, failed)
	if suggested := suggestedOAuthClient(results); suggested != "" {
		fmt.Fprintf(command.OutOrStdout(), "Suggested client from active token: %s\n", suggested)
		if configured := configuredOAuthClient(results); configured != "" && configured != suggested {
			fmt.Fprintf(command.OutOrStdout(), "WARNING: configured client %s does not match the active token client.\n", configured)
		}
	}
	fmt.Fprintln(command.OutOrStdout())
	for _, result := range results {
		printAuthDiagnostic(command, result)
	}
}

func suggestedOAuthClient(results []quota.AuthDiagnostic) string {
	for _, result := range results {
		if !result.Active || result.CurrentTokenInfo == nil {
			continue
		}
		if client := tokenInfoClient(*result.CurrentTokenInfo); client != "" {
			return client
		}
	}
	for _, result := range results {
		if result.CurrentTokenInfo == nil {
			continue
		}
		if client := tokenInfoClient(*result.CurrentTokenInfo); client != "" {
			return client
		}
	}
	return ""
}

func configuredOAuthClient(results []quota.AuthDiagnostic) string {
	for _, result := range results {
		if result.Active && strings.TrimSpace(result.EffectiveClientID) != "" {
			return strings.TrimSpace(result.EffectiveClientID)
		}
	}
	for _, result := range results {
		if strings.TrimSpace(result.EffectiveClientID) != "" {
			return strings.TrimSpace(result.EffectiveClientID)
		}
	}
	return ""
}

func tokenInfoClient(info quota.OAuthTokenInfo) string {
	if client := strings.TrimSpace(info.IssuedTo); client != "" {
		return client
	}
	return strings.TrimSpace(info.Audience)
}

func printAuthDiagnostic(command *cobra.Command, result quota.AuthDiagnostic) {
	marker := "○"
	if result.Active {
		marker = "●"
	}
	identity := result.Profile
	if strings.TrimSpace(result.Email) != "" {
		identity += "  " + result.Email
	}
	fmt.Fprintf(command.OutOrStdout(), "%s %s\n", marker, identity)
	fmt.Fprintf(command.OutOrStdout(), "  saved tokens: access=%s refresh=%s\n", yesNo(result.AccessTokenPresent), yesNo(result.RefreshTokenPresent))
	fmt.Fprintf(command.OutOrStdout(), "  OAuth client: %s", result.ClientIDSource)
	if result.EffectiveClientID != "" {
		fmt.Fprintf(command.OutOrStdout(), " · %s", result.EffectiveClientID)
	}
	fmt.Fprintf(command.OutOrStdout(), " · secret=%s\n", yesNo(result.ClientSecretConfigured))

	if result.CurrentTokenInfo != nil {
		printTokenInfo(command, "current token", *result.CurrentTokenInfo)
	} else if result.CurrentTokenInfoError != "" {
		fmt.Fprintf(command.OutOrStdout(), "  current token: %s\n", result.CurrentTokenInfoError)
	}
	if result.ClientIDMatchesToken != nil {
		fmt.Fprintf(command.OutOrStdout(), "  configured client matches token: %s\n", yesNo(*result.ClientIDMatchesToken))
	}

	fmt.Fprintf(command.OutOrStdout(), "  refresh: %s", result.RefreshStatus)
	if result.RefreshError != "" {
		fmt.Fprintf(command.OutOrStdout(), " · %s", result.RefreshError)
	}
	if result.RefreshHTTPStatus != 0 {
		fmt.Fprintf(command.OutOrStdout(), " · HTTP %d", result.RefreshHTTPStatus)
	}
	fmt.Fprintln(command.OutOrStdout())
	if result.RefreshErrorDescription != "" {
		fmt.Fprintf(command.OutOrStdout(), "    %s\n", result.RefreshErrorDescription)
	}
	if result.RefreshErrorSubtype != "" {
		fmt.Fprintf(command.OutOrStdout(), "    subtype: %s\n", result.RefreshErrorSubtype)
	}
	if result.RefreshedTokenInfo != nil {
		printTokenInfo(command, "refreshed token", *result.RefreshedTokenInfo)
	}

	if result.QuotaStatus != "" {
		fmt.Fprintf(command.OutOrStdout(), "  quota: %s", result.QuotaStatus)
		if result.QuotaModels > 0 {
			fmt.Fprintf(command.OutOrStdout(), " · %d models", result.QuotaModels)
		}
		if result.QuotaSource != "" {
			fmt.Fprintf(command.OutOrStdout(), " · %s", result.QuotaSource)
		}
		if result.QuotaError != "" {
			fmt.Fprintf(command.OutOrStdout(), " · %s", result.QuotaError)
		}
		fmt.Fprintln(command.OutOrStdout())
	}
	if result.Error != "" {
		fmt.Fprintf(command.OutOrStdout(), "  diagnostic error: %s\n", result.Error)
	}
	fmt.Fprintln(command.OutOrStdout())
}

func printTokenInfo(command *cobra.Command, label string, info quota.OAuthTokenInfo) {
	client := tokenInfoClient(info)
	if client != "" {
		fmt.Fprintf(command.OutOrStdout(), "  %s client: %s\n", label, client)
	}
	if len(info.Scopes) > 0 {
		scopes := append([]string(nil), info.Scopes...)
		sort.Strings(scopes)
		fmt.Fprintf(command.OutOrStdout(), "  %s scopes: %s\n", label, strings.Join(scopes, " "))
	}
	if info.ExpiresIn != 0 {
		fmt.Fprintf(command.OutOrStdout(), "  %s expires in: %ds\n", label, info.ExpiresIn)
	}
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}
