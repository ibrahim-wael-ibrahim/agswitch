package keyring

import (
	"context"
	"errors"

	"github.com/ibrahim-wael/agswitch/internal/credentials"
	"github.com/zalando/go-keyring"
)

const (
	DefaultService = "agswitch"
	DefaultUsername = "antigravity"
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
	return &KeyringActiveStore{
		Service:  DefaultService,
		Username: DefaultUsername,
	}
}

func (s *KeyringActiveStore) Load(ctx context.Context) (credentials.Credential, error) {
	service, username := s.identifiers()
	raw, err := keyring.Get(service, username)
	if err != nil {
		return credentials.Credential{}, err
	}

	return credentials.Parse([]byte(raw))
}

func (s *KeyringActiveStore) Save(ctx context.Context, credential credentials.Credential) error {
	service, username := s.identifiers()
	if len(credential.Raw) == 0 {
		return errors.New("credential is empty")
	}

	return keyring.Set(service, username, string(credential.Raw))
}

func (s *KeyringActiveStore) Clear(ctx context.Context) error {
	service, username := s.identifiers()
	return keyring.Delete(service, username)
}

func (s *KeyringActiveStore) identifiers() (string, string) {
	if s == nil {
		return DefaultService, DefaultUsername
	}

	service := s.Service
	if service == "" {
		service = DefaultService
	}

	username := s.Username
	if username == "" {
		username = DefaultUsername
	}

	return service, username
}
