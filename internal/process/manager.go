package process

import (
	"context"
	"errors"
)

type Detector interface {
	Running(ctx context.Context, executable string) (bool, error)
}

type Launcher interface {
	Launch(ctx context.Context, executable string, args ...string) error
}

type Quitter interface {
	Quit(ctx context.Context, executable string) error
}

type BackendReloader interface {
	Reload(ctx context.Context) error
}

type Manager struct {
	Executable string
	Detector   Detector
	Launcher   Launcher
	Quitter    Quitter
	Backend    BackendReloader
}

func NewManager(executable string) *Manager {
	return &Manager{Executable: executable}
}

func (m *Manager) Running(ctx context.Context) (bool, error) {
	if m == nil || m.Detector == nil {
		return false, errors.New("process detector is not configured")
	}
	return m.Detector.Running(ctx, m.Executable)
}

func (m *Manager) Stop(ctx context.Context) error {
	if m == nil || m.Quitter == nil {
		return errors.New("process quitter is not configured")
	}
	return m.Quitter.Quit(ctx, m.Executable)
}

func (m *Manager) Start(ctx context.Context) error {
	if m == nil || m.Launcher == nil {
		return errors.New("process launcher is not configured")
	}
	return m.Launcher.Launch(ctx, m.Executable)
}

func (m *Manager) ReloadBackend(ctx context.Context) error {
	if m == nil || m.Backend == nil {
		return errors.New("language server reloader is not configured")
	}
	return m.Backend.Reload(ctx)
}
