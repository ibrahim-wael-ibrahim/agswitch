package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ibrahim-wael/agswitch/internal/account"
	"github.com/ibrahim-wael/agswitch/internal/credentials"
	"github.com/ibrahim-wael/agswitch/internal/profile"
)

type ProfileInfo struct {
	Account        account.Account `json:"account"`
	Storage        string          `json:"storage"`
	CredentialType string          `json:"credential_type,omitempty"`
}

func (s *Service) Update(ctx context.Context, name string) error {
	if err := profile.Validate(name); err != nil {
		return err
	}
	if s.Active == nil {
		return errors.New("active credential store is not configured")
	}

	const timeout = 20 * time.Second
	const pollInterval = 500 * time.Millisecond
	const expirySkew = time.Minute

	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	var lastStatus credentials.TokenStatus
	for {
		credential, err := s.Active.Load(ctx)
		if err != nil {
			return fmt.Errorf("read active Antigravity credential: %w", err)
		}
		lastStatus = credentials.InspectToken(credential.Raw)
		if lastStatus.Fresh(s.now(), expirySkew) {
			return s.saveCredential(ctx, name, credential, true)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			if lastStatus.ExpiryKnown {
				return fmt.Errorf("active access token did not refresh before timeout; last expiry was %s", lastStatus.Expiry.Format(time.RFC3339))
			}
			return errors.New("active access token did not become usable before timeout")
		case <-ticker.C:
		}
	}
}

func (s *Service) Info(ctx context.Context, name string) (ProfileInfo, error) {
	if err := profile.Validate(name); err != nil {
		return ProfileInfo{}, err
	}
	if s.Profiles == nil || s.Accounts == nil {
		return ProfileInfo{}, errors.New("profile storage is not configured")
	}
	item, err := s.Accounts.Get(ctx, name)
	if err != nil {
		return ProfileInfo{}, err
	}
	credential, err := s.Profiles.Load(ctx, name)
	if err != nil {
		return ProfileInfo{}, err
	}
	return ProfileInfo{
		Account:        item,
		Storage:        "system-keyring",
		CredentialType: credential.CredentialType,
	}, nil
}

func (s *Service) Clone(ctx context.Context, source, target string, force bool) error {
	if err := profile.Validate(source); err != nil {
		return err
	}
	if err := profile.Validate(target); err != nil {
		return err
	}
	if source == target {
		return errors.New("source and target profiles are the same")
	}
	if s.Profiles == nil || s.Accounts == nil {
		return errors.New("profile storage is not configured")
	}
	return s.withLock(ctx, func() error {
		credential, err := s.Profiles.Load(ctx, source)
		if err != nil {
			return fmt.Errorf("load source profile %q: %w", source, err)
		}
		unlocked := *s
		unlocked.Locker = nil
		return unlocked.saveCredential(ctx, target, credential, force)
	})
}

func (s *Service) Rename(ctx context.Context, source, target string, force bool) error {
	if err := profile.Validate(source); err != nil {
		return err
	}
	if err := profile.Validate(target); err != nil {
		return err
	}
	if source == target {
		return nil
	}
	if s.Profiles == nil || s.Accounts == nil {
		return errors.New("profile storage is not configured")
	}
	return s.withLock(ctx, func() error {
		sourceCredential, err := s.Profiles.Load(ctx, source)
		if err != nil {
			return fmt.Errorf("load source profile %q: %w", source, err)
		}
		sourceAccount, err := s.Accounts.Get(ctx, source)
		if err != nil {
			return fmt.Errorf("load source metadata %q: %w", source, err)
		}

		targetCredential, targetCredentialErr := s.Profiles.Load(ctx, target)
		targetCredentialExists := targetCredentialErr == nil
		if targetCredentialErr != nil && !errors.Is(targetCredentialErr, credentials.ErrNotFound) {
			return targetCredentialErr
		}
		targetAccount, targetAccountErr := s.Accounts.Get(ctx, target)
		targetAccountExists := targetAccountErr == nil
		if targetAccountErr != nil && !errors.Is(targetAccountErr, account.ErrNotFound) {
			return targetAccountErr
		}
		if (targetCredentialExists || targetAccountExists) && !force {
			return ErrProfileExists
		}

		rollbackTarget := func() error {
			var rollbackErr error
			if targetCredentialExists {
				rollbackErr = errors.Join(rollbackErr, s.Profiles.Save(ctx, target, targetCredential))
			} else if deleteErr := s.Profiles.Delete(ctx, target); deleteErr != nil && !errors.Is(deleteErr, credentials.ErrNotFound) {
				rollbackErr = errors.Join(rollbackErr, deleteErr)
			}
			if targetAccountExists {
				rollbackErr = errors.Join(rollbackErr, s.Accounts.Save(ctx, targetAccount))
			} else if deleteErr := s.Accounts.Delete(ctx, target); deleteErr != nil && !errors.Is(deleteErr, account.ErrNotFound) {
				rollbackErr = errors.Join(rollbackErr, deleteErr)
			}
			return rollbackErr
		}

		if err := s.Profiles.Save(ctx, target, sourceCredential); err != nil {
			return err
		}
		verified, err := s.Profiles.Load(ctx, target)
		if err != nil || verified.Fingerprint != sourceCredential.Fingerprint {
			if err == nil {
				err = errors.New("renamed profile verification fingerprint mismatch")
			}
			return errors.Join(err, rollbackTarget())
		}
		renamedAccount := sourceAccount
		renamedAccount.ID = target
		if err := s.Accounts.Save(ctx, renamedAccount); err != nil {
			return errors.Join(err, rollbackTarget())
		}
		if err := s.Accounts.Delete(ctx, source); err != nil {
			return errors.Join(err, rollbackTarget())
		}
		if err := s.Profiles.Delete(ctx, source); err != nil {
			restoreErr := s.Accounts.Save(ctx, sourceAccount)
			return errors.Join(err, restoreErr, rollbackTarget())
		}
		return nil
	})
}
