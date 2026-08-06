//go:build linux

package process

import (
	"context"
	"errors"
	"fmt"
	"syscall"
	"time"
)

// LanguageServerReloader restarts only Antigravity's backend. Electron keeps
// the workspace, chat and terminals alive and respawns a fresh language server
// that reads the newly activated keyring credential.
type LanguageServerReloader struct {
	Executable string
	Timeout    time.Duration
	Force      bool
}

func (r LanguageServerReloader) Reload(ctx context.Context) error {
	if r.Executable == "" {
		return errors.New("language server path is empty")
	}
	oldPIDs, err := matchingPIDs(ctx, r.Executable, false)
	if err != nil {
		return err
	}
	if len(oldPIDs) == 0 {
		return errors.New("Antigravity language server is not running")
	}
	old := make(map[int]struct{}, len(oldPIDs))
	for _, pid := range oldPIDs {
		old[pid] = struct{}{}
		if err := syscall.Kill(pid, syscall.SIGTERM); err != nil && !errors.Is(err, syscall.ESRCH) {
			return fmt.Errorf("stop language server pid %d: %w", pid, err)
		}
	}

	timeout := r.Timeout
	if timeout <= 0 {
		timeout = 12 * time.Second
	}
	if err := r.waitForReplacement(ctx, old, timeout); err == nil {
		return nil
	} else if !r.Force {
		return err
	}

	for pid := range old {
		if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
			return fmt.Errorf("force stop language server pid %d: %w", pid, err)
		}
	}
	return r.waitForReplacement(ctx, old, 5*time.Second)
}

func (r LanguageServerReloader) waitForReplacement(ctx context.Context, old map[int]struct{}, timeout time.Duration) error {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		pids, err := matchingPIDs(ctx, r.Executable, false)
		if err != nil {
			return err
		}
		oldStillRunning := false
		newRunning := false
		for _, pid := range pids {
			if _, exists := old[pid]; exists {
				oldStillRunning = true
			} else {
				newRunning = true
			}
		}
		if !oldStillRunning && newRunning {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return errors.New("language server replacement timeout")
		case <-ticker.C:
		}
	}
}
