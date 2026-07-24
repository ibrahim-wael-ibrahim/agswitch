//go:build linux

package process

import (
	"context"
	"os/exec"
)

type LinuxLauncher struct{}

func (LinuxLauncher) Launch(ctx context.Context, executable string, args ...string) error {
	if executable == "" {
		return nil
	}

	cmd := exec.CommandContext(ctx, executable, args...)
	return cmd.Start()
}
