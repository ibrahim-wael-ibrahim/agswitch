package keyring

import (
	"context"
	"errors"

	"github.com/ibrahim-wael/agswitch/internal/credentials"
	zkeyring "github.com/zalando/go-keyring"
)

const (
	ActiveService    = "gemini"
	ActiveUsername   = "antigravity"
	ProfileService   = "agswitch"
	profileKeyPrefix = "profile:"
)

type ActiveStore interface {
	Load(ctx context.Context) (credentials.Credential, error)
	Save(ctx context.Context, credential credentials.Credential) error
	Clear(ctx context.Context) error
}

type KeyringActiveStore struct {
	Service  string
	Username string
}

func NewActiveStore() *KeyringActiveStore {
	return &KeyringActiveStore{Service: ActiveService, Username: ActiveUsername}
}

func (s *KeyringActiveStore) Load(ctx context.Context) (credentials.Credential, error) {
	if err := ctx.Err(); err != nil {
		return credentials.Credential{}, err
	}
	service, username := s.identifiers()
	raw, err := zkeyring.Get(service, username)
	if errors.Is(err, zkeyring.ErrNotFound) {
		return credentials.Credential{}, credentials.ErrNotFound
	}
	if err != nil {
		return credentials.Credential{}, err
	}
	return credentials.Parse([]byte(raw))
}

func (s *KeyringActiveStore) Save(ctx context.Context, credential credentials.Credential) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(credential.Raw) == 0 {
		return errors.New("credential is empty")
	}
	if _, err := credentials.Parse(credential.Raw); err != nil {
		return err
	}
	service, username := s.identifiers()
	return zkeyring.Set(service, username, string(credential.Raw))
}

func (s *KeyringActiveStore) Clear(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	service, username := s.identifiers()
	err := zkeyring.Delete(service, username)
	if errors.Is(err, zkeyring.ErrNotFound) {
		return nil
	}
	return err
}

func (s *KeyringActiveStore) identifiers() (string, string) {
	if s == nil {
		return ActiveService, ActiveUsername
	}
	service := s.Service
	if service == "" {
		service = ActiveService
	}
	username := s.Username
	if username == "" {
		username = ActiveUsername
	}
	return service, username
}
