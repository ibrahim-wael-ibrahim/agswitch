package fzfui

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ibrahim-wael/agswitch/internal/account"
	"github.com/ibrahim-wael/agswitch/internal/autoswitch"
	"github.com/ibrahim-wael/agswitch/internal/brand"
	"github.com/ibrahim-wael/agswitch/internal/quota"
	"github.com/ibrahim-wael/agswitch/internal/switcher"
)

type AccountBackend interface {
	List(context.Context) ([]account.Account, error)
	Use(context.Context, string, switcher.Options) error
}

type QuotaBackend interface {
	FetchAll(context.Context, []account.Account, bool) []quota.Result
}

type Options struct {
	Stay          bool
	Version       string
	AutoThreshold int
	Stdin         io.Reader
	Stdout        io.Writer
	Stderr        io.Writer
}

type row struct {
	Preview string
	Profile string
	Display string
}

func Run(ctx context.Context, accountsBackend AccountBackend, quotaBackend QuotaBackend, options Options) error {
	if _, err := exec.LookPath("fzf"); err != nil {
		return errors.New("fzf is required for the dashboard; install it and run agswitch again")
	}
	if accountsBackend == nil || quotaBackend == nil {
		return errors.New("dashboard backend is not configured")
	}
	if options.Stdout == nil {
		options.Stdout = os.Stdout
	}
	if options.Stderr == nil {
		options.Stderr = os.Stderr
	}
	if options.AutoThreshold <= 0 {
		options.AutoThreshold = 20
	}

	forceRefresh := false
	for {
		clearScreen(options.Stdout)
		fmt.Fprint(options.Stdout, brand.Banner(options.Version))
		message := "Loading account quotas…"
		if forceRefresh {
			message = "Refreshing live account quotas…"
		}
		fmt.Fprintf(options.Stdout, "%s%s%s\n", brand.Magenta, message, brand.Reset)

		accounts, err := accountsBackend.List(ctx)
		if err != nil {
			return err
		}
		if len(accounts) == 0 {
			return errors.New("no saved profiles; run agswitch migrate or agswitch save <profile>")
		}
		results := quotaBackend.FetchAll(ctx, accounts, forceRefresh)
		forceRefresh = false
		current := currentProfile(accounts)
		decision := autoswitch.Select(results, current, options.AutoThreshold)

		selected, key, err := choose(ctx, accounts, results, decision, options)
		if err != nil {
			return err
		}
		if selected == "" {
			clearScreen(options.Stdout)
			return nil
		}
		if key == "ctrl-r" {
			forceRefresh = true
			continue
		}
		if key == "ctrl-a" {
			if !decision.Switch || decision.Selected.Profile == "" {
				clearScreen(options.Stdout)
				fmt.Fprintf(options.Stdout, "%sNo auto-switch needed:%s %s\n", brand.Yellow, brand.Reset, decision.Reason)
				return nil
			}
			selected = decision.Selected.Profile
		}

		clearScreen(options.Stdout)
		fmt.Fprintf(options.Stdout, "%sSwitching to %s…%s\n", brand.Magenta, selected, brand.Reset)
		if err := accountsBackend.Use(ctx, selected, switcher.Options{LaunchMode: switcher.AlwaysLaunch}); err != nil {
			return err
		}
		fmt.Fprintf(options.Stdout, "%sStarted Antigravity with %s%s\n", brand.Green, selected, brand.Reset)
		if !options.Stay {
			return nil
		}
	}
}

func choose(ctx context.Context, accounts []account.Account, results []quota.Result, decision autoswitch.Decision, options Options) (string, string, error) {
	dir, err := os.MkdirTemp("", "agswitch-dashboard-*")
	if err != nil {
		return "", "", err
	}
	defer os.RemoveAll(dir)
	if err := os.Chmod(dir, 0o700); err != nil {
		return "", "", err
	}
	rows, err := buildRows(dir, accounts, results, decision, options)
	if err != nil {
		return "", "", err
	}
	var input strings.Builder
	for _, item := range rows {
		fmt.Fprintf(&input, "%s\t%s\t%s\n", item.Preview, item.Profile, item.Display)
	}
	header := fmt.Sprintf("ENTER switch  •  CTRL-A auto (%d%%)  •  CTRL-R refresh  •  ESC quit", decision.Threshold)
	args := []string{
		"--ansi", "--no-multi", "--layout=reverse", "--border=rounded", "--height=100%",
		"--info=inline-right", "--prompt=  profile › ", "--pointer=◆", "--marker=✓",
		"--header=" + header, "--header-first", "--border-label= agswitch dashboard ",
		"--delimiter=\t", "--with-nth=3..", "--nth=2,3", "--preview=cat -- {1}",
		"--preview-window=right,62%,border-left,wrap", "--expect=enter,ctrl-r,ctrl-a",
		"--color=fg:#cdd6f4,bg:#11111b,hl:#89b4fa,fg+:#ffffff,bg+:#313244,hl+:#89dceb,pointer:#f5c2e7,marker:#a6e3a1,prompt:#cba6f7,spinner:#f9e2af,header:#94e2d5,border:#585b70,label:#89dceb",
	}
	command := exec.CommandContext(ctx, "fzf", args...)
	command.Stdin = strings.NewReader(input.String())
	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = options.Stderr
	if err := command.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && (exitErr.ExitCode() == 1 || exitErr.ExitCode() == 130) {
			return "", "", nil
		}
		return "", "", fmt.Errorf("run fzf: %w", err)
	}
	lines := strings.Split(strings.TrimRight(output.String(), "\n"), "\n")
	if len(lines) < 2 {
		return "", "", nil
	}
	key := strings.TrimSpace(lines[0])
	fields := strings.SplitN(lines[1], "\t", 3)
	if len(fields) < 2 {
		return "", "", errors.New("fzf returned an invalid selection")
	}
	return strings.TrimSpace(fields[1]), key, nil
}

func buildRows(dir string, accounts []account.Account, results []quota.Result, decision autoswitch.Decision, options Options) ([]row, error) {
	byProfile := make(map[string]quota.Result, len(results))
	for _, result := range results {
		byProfile[result.Profile] = result
	}
	rows := make([]row, 0, len(accounts))
	for index, item := range accounts {
		result := byProfile[item.ID]
		preview := filepath.Join(dir, fmt.Sprintf("%03d.txt", index))
		if err := os.WriteFile(preview, []byte(renderPreview(item, result, decision, options)), 0o600); err != nil {
			return nil, err
		}
		rows = append(rows, row{Preview: preview, Profile: item.ID, Display: renderRow(item, result, decision)})
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Profile < rows[j].Profile })
	return rows, nil
}

func renderRow(item account.Account, result quota.Result, decision autoswitch.Decision) string {
	active := "  "
	if item.Active {
		active = "\033[1;32m●\033[0m "
	}
	recommended := ""
	if decision.Switch && decision.Selected.Profile == item.ID {
		recommended = "  \033[1;35mAUTO PICK\033[0m"
	}
	email := strings.TrimSpace(item.Email)
	if email == "" {
		email = item.ID
	}
	if result.Err != nil {
		return fmt.Sprintf("%s\033[1;37m%-32s\033[0m  \033[31munavailable\033[0m%s  \033[2m%s\033[0m", active, email, recommended, item.ID)
	}
	minimum, known := quota.MinimumKnownRemaining(result.Snapshot)
	health := "unknown"
	if known {
		health = colorPercent(minimum) + " min"
	}
	cached := ""
	if result.Snapshot.Cached || result.Snapshot.Source == "cache-stale" {
		cached = "  \033[2mcached\033[0m"
	}
	return fmt.Sprintf("%s\033[1;37m%-32s\033[0m  %s%s%s  \033[2m%s\033[0m", active, email, health, cached, recommended, item.ID)
}

func renderPreview(item account.Account, result quota.Result, decision autoswitch.Decision, options Options) string {
	var output strings.Builder
	fmt.Fprint(&output, brand.Banner(options.Version))
	email := strings.TrimSpace(item.Email)
	if email == "" {
		email = item.ID
	}
	fmt.Fprintf(&output, "%s%s%s\n", brand.White, email, brand.Reset)
	fmt.Fprintf(&output, "%sPROFILE%s  %s\n", brand.Muted, brand.Reset, item.ID)
	if item.Active {
		fmt.Fprintf(&output, "%sSTATUS%s   %sACTIVE%s\n", brand.Muted, brand.Reset, brand.Green, brand.Reset)
	}
	if decision.Switch && decision.Selected.Profile == item.ID {
		fmt.Fprintf(&output, "%sAUTO%s     Recommended at ≤ %d%%\n", brand.Muted, brand.Reset, decision.Threshold)
	}
	if result.Err != nil {
		fmt.Fprintf(&output, "\n%sQuota unavailable%s\n%s\n", brand.Red, brand.Reset, result.Err)
		return output.String()
	}
	snapshot := result.Snapshot
	fmt.Fprintf(&output, "%sPLAN%s     %s\n", brand.Muted, brand.Reset, valueOr(snapshot.SubscriptionTier, "unknown"))
	fmt.Fprintf(&output, "%sSOURCE%s   %s\n", brand.Muted, brand.Reset, snapshot.Source)
	if !snapshot.FetchedAt.IsZero() {
		fmt.Fprintf(&output, "%sUPDATED%s  %s\n", brand.Muted, brand.Reset, snapshot.FetchedAt.Local().Format("2006-01-02 15:04:05"))
	}
	if warning := snapshot.Metadata["warning"]; warning != "" {
		fmt.Fprintf(&output, "%sWARNING%s  %s\n", brand.Yellow, brand.Reset, warning)
	}
	fmt.Fprintln(&output, "\n\033[2m────────────────────────────────────────────────────────\033[0m")
	models := quota.SortedModels(snapshot)
	if len(models) == 0 {
		fmt.Fprintln(&output, "No model quota returned.")
		return output.String()
	}
	now := time.Now()
	for _, model := range models {
		variant := ""
		if model.Variants > 1 {
			variant = fmt.Sprintf("  %s%d variants%s", brand.Muted, model.Variants, brand.Reset)
		}
		fmt.Fprintf(&output, "\n%s%s%s%s\n", brand.White, model.Name, brand.Reset, variant)
		fmt.Fprintf(&output, "%s  %s", progressBar(model.Remaining), colorPercent(model.Remaining))
		if reset := quota.ResetIn(model.ResetAt, now); reset > 0 {
			fmt.Fprintf(&output, "  %sreset in %s%s", brand.Muted, compactDuration(reset), brand.Reset)
		}
		fmt.Fprintln(&output)
	}
	fmt.Fprintf(&output, "\n%sENTER switch · CTRL-A auto-pick · CTRL-R refresh%s\n", brand.Muted, brand.Reset)
	return output.String()
}

func currentProfile(accounts []account.Account) string {
	for _, item := range accounts {
		if item.Active {
			return item.ID
		}
	}
	return ""
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func colorPercent(value int) string {
	if value < 0 {
		return "\033[2munknown\033[0m"
	}
	color := "32"
	if value <= 20 {
		color = "31"
	} else if value < 60 {
		color = "33"
	}
	return fmt.Sprintf("\033[1;%sm%d%%\033[0m", color, value)
}

func progressBar(value int) string {
	if value < 0 {
		return "\033[2m[····················]\033[0m"
	}
	if value > 100 {
		value = 100
	}
	filled := value / 5
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", 20-filled) + "]"
}

func compactDuration(value time.Duration) string {
	if value >= 24*time.Hour {
		return fmt.Sprintf("%dd %dh", int(value/(24*time.Hour)), int(value%(24*time.Hour)/time.Hour))
	}
	if value >= time.Hour {
		hours := int(value / time.Hour)
		minutes := int(value % time.Hour / time.Minute)
		if minutes == 0 {
			return fmt.Sprintf("%dh", hours)
		}
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", int(value/time.Minute))
}

func clearScreen(output io.Writer) {
	fmt.Fprint(output, "\033[2J\033[H")
}
