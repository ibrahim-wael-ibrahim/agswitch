//go:build linux

package process

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type SignalQuitter struct {
	Detector Detector
	Timeout  time.Duration
	Force    bool
}

func (q SignalQuitter) Quit(ctx context.Context, executable string) error {
	if q.Detector == nil {
		q.Detector = LinuxDetector{}
	}
	if q.Timeout <= 0 {
		q.Timeout = 8 * time.Second
	}
	pids, err := matchingPIDs(ctx, executable, true)
	if err != nil {
		return err
	}
	if len(pids) == 0 {
		return nil
	}
	if err := syscall.Kill(pids[0], syscall.SIGTERM); err != nil && !errors.Is(err, syscall.ESRCH) {
		return fmt.Errorf("send SIGTERM: %w", err)
	}
	stopped, err := waitForRunning(ctx, q.Detector, executable, false, q.Timeout)
	if err != nil {
		return err
	}
	if stopped {
		return nil
	}
	if !q.Force {
		return errors.New("Antigravity did not quit gracefully")
	}
	pids, err = matchingPIDs(ctx, executable, false)
	if err != nil {
		return err
	}
	for _, pid := range pids {
		if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
			return fmt.Errorf("force close pid %d: %w", pid, err)
		}
	}
	stopped, err = waitForRunning(ctx, q.Detector, executable, false, 3*time.Second)
	if err != nil {
		return err
	}
	if !stopped {
		return errors.New("Antigravity is still running after force close")
	}
	return nil
}

func matchingPIDs(ctx context.Context, executable string, oldestOnly bool) ([]int, error) {
	args := []string{"-f"}
	if oldestOnly {
		args = append(args, "-o")
	}
	args = append(args, "--", executable)
	output, err := exec.CommandContext(ctx, "pgrep", args...).Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return nil, nil
		}
		return nil, fmt.Errorf("find Antigravity processes: %w", err)
	}
	fields := strings.Fields(string(output))
	pids := make([]int, 0, len(fields))
	for _, field := range fields {
		pid, err := strconv.Atoi(field)
		if err != nil {
			continue
		}
		pids = append(pids, pid)
	}
	return pids, nil
}

func waitForRunning(ctx context.Context, detector Detector, executable string, expected bool, timeout time.Duration) (bool, error) {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		running, err := detector.Running(ctx, executable)
		if err != nil {
			return false, err
		}
		if running == expected {
			return true, nil
		}
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-deadline.C:
			return false, nil
		case <-ticker.C:
		}
	}
}
