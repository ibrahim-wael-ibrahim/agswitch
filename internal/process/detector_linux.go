//go:build linux

package process

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type LinuxDetector struct{}

func (LinuxDetector) Running(ctx context.Context, executable string) (bool, error) {
	if strings.TrimSpace(executable) == "" {
		return false, errors.New("executable path is empty")
	}
	cmd := exec.CommandContext(ctx, "pgrep", "-f", "--", executable)
	output, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(output)) != "", nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return false, nil
	}
	return false, fmt.Errorf("detect process: %w", err)
}
