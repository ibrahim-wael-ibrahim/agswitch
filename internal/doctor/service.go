package doctor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ibrahim-wael/agswitch/internal/account"
	"github.com/ibrahim-wael/agswitch/internal/config"
	"github.com/ibrahim-wael/agswitch/internal/credentials"
	agsprocess "github.com/ibrahim-wael/agswitch/internal/process"
)

type Status string

const (
	OK Status = "OK"
	Warn Status = "WARN"
	Fail Status = "FAIL"
)

type Check struct { Status Status; Name, Details string }
type ActiveStore interface { Load(context.Context) (credentials.Credential, error) }
type AccountRepository interface { List(context.Context) ([]account.Account, error) }
type ProcessManager interface { Running(context.Context) (bool, error) }
type Service struct { Config config.Config; Active ActiveStore; Accounts AccountRepository; Process ProcessManager }

func (s Service) Run(ctx context.Context) []Check {
	checks := []Check{}
	if runtime.GOOS == "linux" { checks = append(checks, Check{OK, "Operating system", runtime.GOOS}) } else { checks = append(checks, Check{Fail, "Operating system", "Linux required"}) }
	for _, command := range []struct{name string; required bool}{{"pgrep", true}, {"secret-tool", true}, {"busctl", false}, {"fzf", false}} {
		if path, err := exec.LookPath(command.name); err == nil { checks = append(checks, Check{OK, command.name+" installed", path}) } else if command.required { checks = append(checks, Check{Fail, command.name+" installed", "not found"}) } else { checks = append(checks, Check{Warn, command.name+" installed", "not found"}) }
	}
	checks = append(checks,
		checkDir("Config directory", s.Config.BaseDir),
		checkDir("State directory", s.Config.StateDir),
		checkDir("Cache directory", s.Config.CacheDir),
		checkPrivateFile("Account metadata", s.Config.AccountsPath),
		checkWritableParent("Log destination", s.Config.LogPath),
		checkWritableParent("Quota cache", s.Config.QuotaCache),
		checkExecutable(s.Config.AppPath),
	)
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" { checks = append(checks, Check{Warn, "D-Bus session", "not set"}) } else { checks = append(checks, Check{OK, "D-Bus session", "available"}) }
	if available, err := agsprocess.DBusTrayAvailable(ctx); err != nil { checks = append(checks, Check{Warn, "Tray D-Bus services", err.Error()}) } else if available { checks = append(checks, Check{OK, "Tray D-Bus services", "detected; Antigravity Quit action still requires device verification"}) } else { checks = append(checks, Check{Warn, "Tray D-Bus services", "not detected; signal fallback will be used"}) }
	if s.Active != nil {
		if credential, err := s.Active.Load(ctx); err == nil { detail := "credential found"; if credential.Email != "" { detail = credential.Email }; checks = append(checks, Check{OK, "Active credential", detail}) } else if errors.Is(err, credentials.ErrNotFound) { checks = append(checks, Check{Warn, "Active credential", "not found"}) } else { checks = append(checks, Check{Fail, "Active credential", err.Error()}) }
	}
	if s.Accounts != nil {
		if items, err := s.Accounts.List(ctx); err == nil { checks = append(checks, Check{OK, "Saved profiles", fmt.Sprintf("%d profile(s)", len(items))}) } else { checks = append(checks, Check{Fail, "Saved profiles", err.Error()}) }
	}
	if s.Process != nil {
		if running, err := s.Process.Running(ctx); err != nil { checks = append(checks, Check{Fail, "Antigravity process", err.Error()}) } else if running { checks = append(checks, Check{OK, "Antigravity process", "running"}) } else { checks = append(checks, Check{Warn, "Antigravity process", "stopped"}) }
	}
	return checks
}

func HasFailures(checks []Check) bool { for _, check := range checks { if check.Status == Fail { return true } }; return false }

func checkDir(name, path string) Check {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) { return Check{Warn, name, "will be created: "+path} }
	if err != nil { return Check{Fail, name, err.Error()} }
	if !info.IsDir() { return Check{Fail, name, "not a directory"} }
	if info.Mode().Perm()&0o077 != 0 { return Check{Warn, name, fmt.Sprintf("permissions are %o; expected 700", info.Mode().Perm())} }
	return Check{OK, name, path}
}

func checkPrivateFile(name, path string) Check {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) { return Check{Warn, name, "not created yet"} }
	if err != nil { return Check{Fail, name, err.Error()} }
	if info.Mode()&os.ModeSymlink != 0 { return Check{Fail, name, "must not be a symbolic link"} }
	if info.IsDir() { return Check{Fail, name, "is a directory"} }
	if info.Mode().Perm()&0o077 != 0 { return Check{Warn, name, fmt.Sprintf("permissions are %o; expected 600", info.Mode().Perm())} }
	return Check{OK, name, filepath.Clean(path)}
}

func checkWritableParent(name, path string) Check {
	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0o700); err != nil { return Check{Fail, name, err.Error()} }
	if err := os.Chmod(parent, 0o700); err != nil { return Check{Warn, name, err.Error()} }
	probe, err := os.CreateTemp(parent, ".agswitch-doctor-*")
	if err != nil { return Check{Fail, name, "not writable: "+err.Error()} }
	probePath := probe.Name(); _ = probe.Close(); _ = os.Remove(probePath)
	return Check{OK, name, path}
}

func checkExecutable(path string) Check {
	info, err := os.Stat(path)
	if err != nil { return Check{Fail, "Antigravity executable", err.Error()} }
	if info.IsDir() || info.Mode().Perm()&0o111 == 0 { return Check{Fail, "Antigravity executable", "not executable: "+path} }
	return Check{OK, "Antigravity executable", strings.TrimSpace(path)}
}
