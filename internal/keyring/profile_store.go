package keyring

import (
	"context"
	"errors"
	"fmt"

	"github.com/ibrahim-wael/agswitch/internal/credentials"
	"github.com/ibrahim-wael/agswitch/internal/profile"
	zkeyring "github.com/zalando/go-keyring"
)

type ProfileStore interface {
	Load(ctx context.Context, name string) (credentials.Credential, error)
	Save(ctx context.Context, name string, credential credentials.Credential) error
	Delete(ctx context.Context, name string) error
}

type KeyringProfileStore struct {
	Service string
}

func NewProfileStore() *KeyringProfileStore {
	return &KeyringProfileStore{Service: ProfileService}
}

func (s *KeyringProfileStore) Load(ctx context.Context, name string) (credentials.Credential, error) {
	if err := ctx.Err(); err != nil {
		return credentials.Credential{}, err
	}
	if err := profile.Validate(name); err != nil {
		return credentials.Credential{}, err
	}
	raw, err := zkeyring.Get(s.service(), profileKey(name))
	if errors.Is(err, zkeyring.ErrNotFound) {
		return credentials.Credential{}, credentials.ErrNotFound
	}
	if err != nil {
		return credentials.Credential{}, err
	}
	return credentials.Parse([]byte(raw))
}

func (s *KeyringProfileStore) Save(ctx context.Context, name string, credential credentials.Credential) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := profile.Validate(name); err != nil {
		return err
	}
	parsed, err := credentials.Parse(credential.Raw)
	if err != nil {
		return err
	}
	return zkeyring.Set(s.service(), profileKey(name), string(parsed.Raw))
}

func (s *KeyringProfileStore) Delete(ctx context.Context, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := profile.Validate(name); err != nil {
		return err
	}
	err := zkeyring.Delete(s.service(), profileKey(name))
	if errors.Is(err, zkeyring.ErrNotFound) {
		return credentials.ErrNotFound
	}
	return err
}

func (s *KeyringProfileStore) service() string {
	if s == nil || s.Service == "" {
		return ProfileService
	}
	return s.Service
}

func profileKey(name string) string {
	return fmt.Sprintf("%s%s", profileKeyPrefix, name)
}
