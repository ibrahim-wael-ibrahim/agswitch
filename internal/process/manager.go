package process

import "context"

type Detector interface {
	Running(ctx context.Context, executable string) (bool, error)
}

type Launcher interface {
	Launch(ctx context.Context, executable string, args ...string) error
}

type Quitter interface {
	Quit(ctx context.Context, executable string) error
}

type Manager struct {
	Executable string
	Detector   Detector
	Launcher   Launcher
	Quitter    Quitter
}

func NewManager(executable string) *Manager {
	return &Manager{Executable: executable}
}

func (m *Manager) Running(ctx context.Context) (bool, error) {
	if m == nil || m.Detector == nil {
		return false, nil
	}

	return m.Detector.Running(ctx, m.Executable)
}

func (m *Manager) Stop(ctx context.Context) error {
	if m == nil || m.Quitter == nil {
		return nil
	}

	return m.Quitter.Quit(ctx, m.Executable)
}

func (m *Manager) Start(ctx context.Context) error {
	if m == nil || m.Launcher == nil {
		return nil
	}

	return m.Launcher.Launch(ctx, m.Executable)
}
