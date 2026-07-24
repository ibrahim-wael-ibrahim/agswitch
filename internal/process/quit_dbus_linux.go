//go:build linux

package process

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
)

// DBusTrayAvailable reports whether the desktop session exposes the basic
// services needed for a future StatusNotifierItem quit implementation. It does
// not claim that Antigravity publishes a compatible Quit action.
func DBusTrayAvailable(ctx context.Context) (bool, error) {
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" {
		return false, nil
	}
	output, err := exec.CommandContext(ctx, "busctl", "--user", "--no-pager", "list").Output()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	text := strings.ToLower(string(output))
	return strings.Contains(text, "statusnotifier") || strings.Contains(text, "dbusmenu"), nil
}
