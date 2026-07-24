//go:build linux

package process

import (
	"context"
	"os/exec"
	"strings"
)

type LinuxDetector struct{}

func (LinuxDetector) Running(ctx context.Context, executable string) (bool, error) {
	if executable == "" {
		return false, nil
	}

	cmd := exec.CommandContext(ctx, "pgrep", "-f", executable)
	output, err := cmd.Output()
	if err != nil {
		return false, nil
	}

	return strings.TrimSpace(string(output)) != "", nil
}
