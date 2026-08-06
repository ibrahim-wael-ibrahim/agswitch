//go:build linux

package process

import (
	"context"
	"errors"
	"fmt"
	"syscall"
	"time"
)

// CleanupQuitter runs the normal application shutdown, then ensures related
// backend executables are gone. This prevents old authenticated language
// servers from surviving a full account switch as orphan processes.
type CleanupQuitter struct {
	Primary Quitter
	Related []string
	Timeout time.Duration
	Force   bool
}

func (q CleanupQuitter) Quit(ctx context.Context, executable string) error {
	if q.Primary == nil {
		return errors.New("primary process quitter is not configured")
	}
	if err := q.Primary.Quit(ctx, executable); err != nil {
		return err
	}
	for _, related := range q.Related {
		if related == "" {
			continue
		}
		if err := stopAllMatching(ctx, related, q.Timeout, q.Force); err != nil {
			return fmt.Errorf("clean related process %q: %w", related, err)
		}
	}
	return nil
}

func stopAllMatching(ctx context.Context, executable string, timeout time.Duration, force bool) error {
	pids, err := matchingPIDs(ctx, executable, false)
	if err != nil || len(pids) == 0 {
		return err
	}
	for _, pid := range pids {
		if err := syscall.Kill(pid, syscall.SIGTERM); err != nil && !errors.Is(err, syscall.ESRCH) {
			return fmt.Errorf("send SIGTERM to pid %d: %w", pid, err)
		}
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	if waitErr := waitForNoMatches(ctx, executable, timeout); waitErr == nil {
		return nil
	} else if !force {
		return waitErr
	}
	pids, err = matchingPIDs(ctx, executable, false)
	if err != nil {
		return err
	}
	for _, pid := range pids {
		if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
			return fmt.Errorf("send SIGKILL to pid %d: %w", pid, err)
		}
	}
	return waitForNoMatches(ctx, executable, 3*time.Second)
}

func waitForNoMatches(ctx context.Context, executable string, timeout time.Duration) error {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		pids, err := matchingPIDs(ctx, executable, false)
		if err != nil {
			return err
		}
		if len(pids) == 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return errors.New("related process cleanup timeout")
		case <-ticker.C:
		}
	}
}
