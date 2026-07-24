package quota

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ibrahim-wael/agswitch/internal/account"
	"github.com/ibrahim-wael/agswitch/internal/credentials"
)

type ProfileStore interface {
	Load(context.Context, string) (credentials.Credential, error)
}

type Service struct {
	Profiles    ProfileStore
	Provider    Provider
	Cache       *FileCache
	Concurrency int
	Now         func() time.Time
}

func (s *Service) Fetch(ctx context.Context, profile string, refresh bool) Result {
	result := Result{Profile: profile}
	now := time.Now().UTC()
	if s.Now != nil {
		now = s.Now().UTC()
	}
	if !refresh && s.Cache != nil {
		if cached, fresh, err := s.Cache.Fresh(profile, now); err == nil && fresh {
			cached.Profile = profile
			result.Snapshot = cached
			return result
		}
	}
	if s.Profiles == nil || s.Provider == nil {
		result.Err = fmt.Errorf("quota service is not configured")
		return result
	}
	credential, err := s.Profiles.Load(ctx, profile)
	if err != nil {
		result.Err = fmt.Errorf("load profile %q for quota: %w", profile, err)
		return result
	}
	snapshot, err := s.Provider.Fetch(ctx, credential)
	if err != nil {
		if s.Cache != nil {
			if cached, ok, cacheErr := s.Cache.Load(profile); cacheErr == nil && ok {
				if cached.Metadata == nil {
					cached.Metadata = map[string]string{}
				}
				cached.Metadata["warning"] = err.Error()
				cached.Source = "cache-stale"
				cached.Profile = profile
				result.Snapshot = cached
				return result
			}
		}
		result.Err = err
		return result
	}
	snapshot.Profile = profile
	if snapshot.Email == "" {
		snapshot.Email = credential.Email
	}
	if s.Cache != nil {
		if cacheErr := s.Cache.Save(profile, snapshot); cacheErr != nil {
			if snapshot.Metadata == nil {
				snapshot.Metadata = map[string]string{}
			}
			snapshot.Metadata["cache_warning"] = cacheErr.Error()
		}
	}
	result.Snapshot = snapshot
	return result
}

func (s *Service) FetchAll(ctx context.Context, accounts []account.Account, refresh bool) []Result {
	results := make([]Result, len(accounts))
	limit := s.Concurrency
	if limit <= 0 {
		limit = 4
	}
	semaphore := make(chan struct{}, limit)
	var wait sync.WaitGroup
	for index, item := range accounts {
		index, item := index, item
		wait.Add(1)
		go func() {
			defer wait.Done()
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				results[index] = Result{Profile: item.ID, Err: ctx.Err()}
				return
			}
			results[index] = s.Fetch(ctx, item.ID, refresh)
			if results[index].Snapshot.Email == "" {
				results[index].Snapshot.Email = item.Email
			}
		}()
	}
	wait.Wait()
	return results
}
