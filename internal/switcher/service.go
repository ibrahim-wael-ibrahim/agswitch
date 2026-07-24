package switcher

import (
	"context"

	"github.com/ibrahim-wael/agswitch/internal/credentials"
)

type ActiveStore interface {
	Load(ctx context.Context) (credentials.Credential, error)
	Save(ctx context.Context, credential credentials.Credential) error
	Clear(ctx context.Context) error
}

type ProfileStore interface {
	Load(ctx context.Context, profile string) (credentials.Credential, error)
	Save(ctx context.Context, profile string, credential credentials.Credential) error
	Delete(ctx context.Context, profile string) error
}

type ProcessManager interface {
	Stop(ctx context.Context) error
	Start(ctx context.Context) error
	Running(ctx context.Context) (bool, error)
}

type Service struct {
	ActiveStore  ActiveStore
	ProfileStore ProfileStore
	Process      ProcessManager
}

func (s Service) Switch(ctx context.Context, profile string) error {
	return nil
}
