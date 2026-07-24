//go:build linux

package process

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/ibrahim-wael/agswitch/internal/fsutil"
)

type LinuxLauncher struct {
	LogPath     string
	MaxLogBytes int64
}

func (l LinuxLauncher) Launch(ctx context.Context, executable string, args ...string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	info, err := os.Stat(executable)
	if err != nil {
		return fmt.Errorf("stat Antigravity executable: %w", err)
	}
	if info.IsDir() || info.Mode().Perm()&0o111 == 0 {
		return errors.New("Antigravity executable is not executable")
	}

	var output io.WriteCloser
	if l.LogPath != "" {
		if err := fsutil.EnsurePrivateDir(filepath.Dir(l.LogPath)); err != nil {
			return err
		}
		if err := l.rotateLog(); err != nil {
			return err
		}
		file, err := os.OpenFile(l.LogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		output = file
	} else {
		file, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		if err != nil {
			return err
		}
		output = file
	}

	cmd := exec.Command(executable, args...)
	cmd.Stdin = nil
	cmd.Stdout = output
	cmd.Stderr = output
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		_ = output.Close()
		return fmt.Errorf("start Antigravity: %w", err)
	}
	go func() {
		_ = cmd.Wait()
		_ = output.Close()
	}()
	return nil
}

func (l LinuxLauncher) rotateLog() error {
	maxBytes := l.MaxLogBytes
	if maxBytes <= 0 {
		maxBytes = 5 << 20
	}
	info, err := os.Stat(l.LogPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("stat Antigravity log: %w", err)
	}
	if info.Size() <= maxBytes {
		return nil
	}
	backup := l.LogPath + ".old"
	if err := os.Remove(backup); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove old log backup: %w", err)
	}
	if err := os.Rename(l.LogPath, backup); err != nil {
		return fmt.Errorf("rotate Antigravity log: %w", err)
	}
	return os.Chmod(backup, 0o600)
}
