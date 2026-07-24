package state

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/ibrahim-wael/agswitch/internal/fsutil"
	"github.com/ibrahim-wael/agswitch/internal/profile"
)

type Snapshot struct {
	Current   string    `json:"current,omitempty"`
	Previous  string    `json:"previous,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type Store struct {
	Path string
	Now  func() time.Time
}

func New(path string) *Store {
	return &Store{Path: path, Now: time.Now}
}

func (s *Store) Load(ctx context.Context) (Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	data, err := os.ReadFile(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return Snapshot{}, nil
	}
	if err != nil {
		return Snapshot{}, err
	}
	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return Snapshot{}, err
	}
	return snapshot, nil
}

func (s *Store) Commit(ctx context.Context, name string) error {
	if err := profile.Validate(name); err != nil {
		return err
	}
	snapshot, err := s.Load(ctx)
	if err != nil {
		return err
	}
	if snapshot.Current != "" && snapshot.Current != name {
		snapshot.Previous = snapshot.Current
	}
	snapshot.Current = name
	now := time.Now().UTC()
	if s.Now != nil {
		now = s.Now().UTC()
	}
	snapshot.UpdatedAt = now
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(s.Path, append(data, '\n'), 0o600)
}
