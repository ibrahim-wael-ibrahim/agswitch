//go:build linux

package process

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

type CommandThenSignalQuitter struct {
	Command  []string
	Detector Detector
	Fallback SignalQuitter
	Timeout  time.Duration
}

func (q CommandThenSignalQuitter) Quit(ctx context.Context, executable string) error {
	detector := q.Detector
	if detector == nil {
		detector = LinuxDetector{}
	}
	if len(q.Command) > 0 {
		cmd := exec.CommandContext(ctx, q.Command[0], q.Command[1:]...)
		if err := cmd.Run(); err == nil {
			timeout := q.Timeout
			if timeout <= 0 {
				timeout = 4 * time.Second
			}
			stopped, waitErr := waitForRunning(ctx, detector, executable, false, timeout)
			if waitErr != nil {
				return waitErr
			}
			if stopped {
				return nil
			}
		} else if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return ctx.Err()
		}
	}
	fallback := q.Fallback
	fallback.Detector = detector
	if err := fallback.Quit(ctx, executable); err != nil {
		return fmt.Errorf("quit Antigravity: %w", err)
	}
	return nil
}
