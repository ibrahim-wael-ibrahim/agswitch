//go:build linux

package process

import (
	"context"
	"os/exec"
)

type SignalQuitter struct{}

func (SignalQuitter) Quit(ctx context.Context, executable string) error {
	if executable == "" {
		return nil
	}

	return exec.CommandContext(ctx, "pkill", "-TERM", "-f", executable).Run()
}
