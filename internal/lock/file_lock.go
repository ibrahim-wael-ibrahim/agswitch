//go:build linux

package lock

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/ibrahim-wael/agswitch/internal/fsutil"
)

var ErrBusy = errors.New("another agswitch operation is already running")

type Locker interface {
	Lock(ctx context.Context) (func() error, error)
}

type FileLocker struct {
	Path string
}

func New(path string) *FileLocker {
	return &FileLocker{Path: path}
}

func (l *FileLocker) Lock(ctx context.Context) (func() error, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if l == nil || l.Path == "" {
		return nil, errors.New("lock path is empty")
	}
	if err := fsutil.EnsurePrivateDir(filepath.Dir(l.Path)); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(l.Path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = file.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return nil, ErrBusy
		}
		return nil, fmt.Errorf("lock agswitch: %w", err)
	}
	unlocked := false
	return func() error {
		if unlocked {
			return nil
		}
		unlocked = true
		unlockErr := syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		closeErr := file.Close()
		if unlockErr != nil {
			return unlockErr
		}
		return closeErr
	}, nil
}
