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
	Stay   bool
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

type row struct {
	Preview string
	Profile string
	Display string
}

func Run(ctx context.Context, accountsBackend AccountBackend, quotaBackend QuotaBackend, options Options) error {
	if _, err := exec.LookPath("fzf"); err != nil {
		return errors.New("fzf is required for the quota TUI; install it and run agswitch again")
	}
	if accountsBackend == nil || quotaBackend == nil {
		return errors.New("quota TUI backend is not configured")
	}
	if options.Stdout == nil {
		options.Stdout = os.Stdout
	}
	if options.Stderr == nil {
		options.Stderr = os.Stderr
	}
	forceRefresh := false
	for {
		clearScreen(options.Stdout)
		message := "loading account quotas…"
		if forceRefresh {
			message = "refreshing live account quotas…"
		}
		fmt.Fprintf(options.Stdout, "\033[1;36magswitch\033[0m  %s\n", message)
		accounts, err := accountsBackend.List(ctx)
		if err != nil {
			return err
		}
		if len(accounts) == 0 {
			return errors.New("no saved profiles; run agswitch migrate or agswitch save <profile>")
		}
		results := quotaBackend.FetchAll(ctx, accounts, forceRefresh)
		forceRefresh = false
		selected, key, err := choose(ctx, accounts, results, options)
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
		clearScreen(options.Stdout)
		fmt.Fprintf(options.Stdout, "\033[1;35mSwitching to %s…\033[0m\n", selected)
		if err := accountsBackend.Use(ctx, selected, switcher.Options{LaunchMode: switcher.AlwaysLaunch}); err != nil {
			return err
		}
		fmt.Fprintf(options.Stdout, "\033[1;32mStarted Antigravity with %s\033[0m\n", selected)
		if !options.Stay {
			return nil
		}
	}
}

func choose(ctx context.Context, accounts []account.Account, results []quota.Result, options Options) (string, string, error) {
	dir, err := os.MkdirTemp("", "agswitch-quota-*")
	if err != nil {
		return "", "", err
	}
	defer os.RemoveAll(dir)
	if err := os.Chmod(dir, 0o700); err != nil {
		return "", "", err
	}
	rows, err := buildRows(dir, accounts, results)
	if err != nil {
		return "", "", err
	}
	var input strings.Builder
	for _, item := range rows {
		fmt.Fprintf(&input, "%s\t%s\t%s\n", item.Preview, item.Profile, item.Display)
	}
	args := []string{
		"--ansi", "--no-multi", "--layout=reverse", "--border=rounded", "--height=100%",
		"--info=inline-right", "--prompt=  account › ", "--pointer=▶", "--marker=✓",
		"--header=ENTER switch account  •  CTRL-R refresh quota  •  ESC quit", "--header-first",
		"--delimiter=\t", "--with-nth=3..", "--nth=2,3", "--preview=cat -- {1}",
		"--preview-window=right,60%,border-left,wrap", "--expect=enter,ctrl-r",
		"--color=fg:#cdd6f4,bg:#1e1e2e,hl:#89b4fa,fg+:#ffffff,bg+:#313244,hl+:#89dceb,pointer:#f5c2e7,marker:#a6e3a1,prompt:#cba6f7,spinner:#f9e2af,header:#94e2d5,border:#585b70,label:#89dceb",
	}
	command := exec.CommandContext(ctx, "fzf", args...)
	command.Stdin = strings.NewReader(input.String())
	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = options.Stderr
	err = command.Run()
	if err != nil {
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

func buildRows(dir string, accounts []account.Account, results []quota.Result) ([]row, error) {
	byProfile := make(map[string]quota.Result, len(results))
	for _, result := range results {
		byProfile[result.Profile] = result
	}
	rows := make([]row, 0, len(accounts))
	for index, item := range accounts {
		result := byProfile[item.ID]
		preview := filepath.Join(dir, fmt.Sprintf("%03d.txt", index))
		if err := os.WriteFile(preview, []byte(renderPreview(item, result)), 0o600); err != nil {
			return nil, err
		}
		rows = append(rows, row{Preview: preview, Profile: item.ID, Display: renderRow(item, result)})
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Profile < rows[j].Profile })
	return rows, nil
}

func renderRow(item account.Account, result quota.Result) string {
	active := "  "
	if item.Active {
		active = "\033[1;32m●\033[0m "
	}
	email := strings.TrimSpace(item.Email)
	if email == "" {
		email = item.ID
	}
	if result.Err != nil {
		return fmt.Sprintf("%s\033[1;37m%-34s\033[0m  \033[31mquota unavailable\033[0m  \033[2m[%s]\033[0m", active, email, item.ID)
	}
	models := quota.SortedModels(result.Snapshot)
	summary := "no model quota"
	if len(models) > 0 {
		parts := make([]string, 0, 3)
		for _, model := range models {
			if len(parts) == 3 {
				break
			}
			parts = append(parts, shortModelName(model.Name)+" "+colorPercent(model.Remaining))
		}
		summary = strings.Join(parts, "  ")
	}
	cached := ""
	if result.Snapshot.Cached || result.Snapshot.Source == "cache-stale" {
		cached = "  \033[2mcache\033[0m"
	}
	return fmt.Sprintf("%s\033[1;37m%-34s\033[0m  %s%s  \033[2m[%s]\033[0m", active, email, summary, cached, item.ID)
}

func renderPreview(item account.Account, result quota.Result) string {
	var output strings.Builder
	email := strings.TrimSpace(item.Email)
	if email == "" {
		email = item.ID
	}
	fmt.Fprintf(&output, "\033[1;36magswitch quota\033[0m\n\n")
	fmt.Fprintf(&output, "\033[1;37m%s\033[0m\n", email)
	fmt.Fprintf(&output, "profile  %s\n", item.ID)
	if item.Active {
		fmt.Fprintln(&output, "status   \033[1;32mactive\033[0m")
	}
	if result.Err != nil {
		fmt.Fprintf(&output, "\n\033[1;31mQuota unavailable\033[0m\n%s\n", result.Err)
		return output.String()
	}
	snapshot := result.Snapshot
	if snapshot.SubscriptionTier != "" {
		fmt.Fprintf(&output, "plan     %s\n", snapshot.SubscriptionTier)
	}
	fmt.Fprintf(&output, "source   %s\n", snapshot.Source)
	if !snapshot.FetchedAt.IsZero() {
		fmt.Fprintf(&output, "updated  %s\n", snapshot.FetchedAt.Local().Format("2006-01-02 15:04:05"))
	}
	if warning := snapshot.Metadata["warning"]; warning != "" {
		fmt.Fprintf(&output, "warning  \033[1;33m%s\033[0m\n", warning)
	}
	fmt.Fprintln(&output, "\n\033[2m────────────────────────────────────────────────────────\033[0m")
	models := quota.SortedModels(snapshot)
	if len(models) == 0 {
		fmt.Fprintln(&output, "No model quota returned.")
		return output.String()
	}
	now := time.Now()
	for _, model := range models {
		fmt.Fprintf(&output, "\n\033[1;37m%s\033[0m\n", model.Name)
		fmt.Fprintf(&output, "%s  %s", progressBar(model.Remaining), colorPercent(model.Remaining))
		if reset := quota.ResetIn(model.ResetAt, now); reset > 0 {
			fmt.Fprintf(&output, "  \033[2mreset in %s\033[0m", compactDuration(reset))
		}
		fmt.Fprintln(&output)
	}
	fmt.Fprintln(&output, "\n\033[2mENTER switches to this account. CTRL-R refreshes live data.\033[0m")
	return output.String()
}

func shortModelName(value string) string {
	value = strings.ReplaceAll(value, "Gemini ", "G ")
	value = strings.ReplaceAll(value, "Claude ", "C ")
	if len(value) > 18 {
		return value[:18]
	}
	return value
}

func colorPercent(value int) string {
	if value < 0 {
		return "\033[2m--\033[0m"
	}
	color := "32"
	if value < 25 {
		color = "31"
	} else if value < 60 {
		color = "33"
	}
	return fmt.Sprintf("\033[1;%sm%d%%\033[0m", color, value)
}

func progressBar(value int) string {
	if value < 0 {
		return "\033[2m[????????????????????]\033[0m"
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
		return fmt.Sprintf("%dh %dm", int(value/time.Hour), int(value%time.Hour/time.Minute))
	}
	return fmt.Sprintf("%dm", int(value/time.Minute))
}

func clearScreen(output io.Writer) {
	fmt.Fprint(output, "\033[2J\033[H")
}
