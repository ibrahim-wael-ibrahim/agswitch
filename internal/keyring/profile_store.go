package keyring

import (
	"context"
	"fmt"

	"github.com/ibrahim-wael/agswitch/internal/credentials"
	"github.com/zalando/go-keyring"
)

type ProfileStore interface {
	Load(ctx context.Context, profile string) (credentials.Credential, error)
	Save(ctx context.Context, profile string, credential credentials.Credential) error
	Delete(ctx context.Context, profile string) error
	List(ctx context.Context) ([]string, error)
}

type KeyringProfileStore struct {
	Service string
}

func NewProfileStore() *KeyringProfileStore {
	return &KeyringProfileStore{Service: DefaultService}
}

func (s *KeyringProfileStore) Load(ctx context.Context, profile string) (credentials.Credential, error) {
	raw, err := keyring.Get(s.service(), profileKey(profile))
	if err != nil {
		return credentials.Credential{}, err
	}

	return credentials.Parse([]byte(raw))
}

func (s *KeyringProfileStore) Save(ctx context.Context, profile string, credential credentials.Credential) error {
	return keyring.Set(s.service(), profileKey(profile), string(credential.Raw))
}

func (s *KeyringProfileStore) Delete(ctx context.Context, profile string) error {
	return keyring.Delete(s.service(), profileKey(profile))
}

func (s *KeyringProfileStore) List(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (s *KeyringProfileStore) service() string {
	if s == nil || s.Service == "" {
		return DefaultService
	}

	return s.Service
}

func profileKey(profile string) string {
	return fmt.Sprintf("profile:%s", profile)
}
