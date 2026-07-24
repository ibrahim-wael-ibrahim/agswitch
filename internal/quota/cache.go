package quota

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
)

type Cache struct {
	Path string
}

func NewCache(path string) Cache {
	return Cache{Path: path}
}

func (c Cache) Load(ctx context.Context) (Snapshot, error) {
	data, err := os.ReadFile(c.Path)
	if err != nil {
		return Snapshot{}, err
	}

	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return Snapshot{}, err
	}

	return snapshot, nil
}

func (c Cache) Save(ctx context.Context, snapshot Snapshot) error {
	if err := os.MkdirAll(filepath.Dir(c.Path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.Path, data, 0o600)
}
