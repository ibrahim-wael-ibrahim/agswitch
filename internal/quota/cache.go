package quota

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/ibrahim-wael/agswitch/internal/fsutil"
)

type FileCache struct {
	Path string
	TTL  time.Duration
	mu   sync.Mutex
}

type cacheFile struct {
	Version   int                 `json:"version"`
	Snapshots map[string]Snapshot `json:"snapshots"`
}

func (c *FileCache) Load(profile string) (Snapshot, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	file, err := c.read()
	if err != nil {
		return Snapshot{}, false, err
	}
	snapshot, ok := file.Snapshots[profile]
	if !ok {
		return Snapshot{}, false, nil
	}
	snapshot.Cached = true
	return snapshot, true, nil
}

func (c *FileCache) Fresh(profile string, now time.Time) (Snapshot, bool, error) {
	snapshot, ok, err := c.Load(profile)
	if err != nil || !ok {
		return snapshot, ok, err
	}
	ttl := c.TTL
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return snapshot, now.Sub(snapshot.FetchedAt) <= ttl, nil
}

func (c *FileCache) Save(profile string, snapshot Snapshot) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	file, err := c.read()
	if err != nil {
		return err
	}
	if file.Snapshots == nil {
		file.Snapshots = map[string]Snapshot{}
	}
	snapshot.Cached = false
	file.Snapshots[profile] = snapshot
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return fsutil.WriteFileAtomic(c.Path, data, 0o600)
}

func (c *FileCache) read() (cacheFile, error) {
	file := cacheFile{Version: 1, Snapshots: map[string]Snapshot{}}
	if c == nil || c.Path == "" {
		return file, nil
	}
	data, err := os.ReadFile(c.Path)
	if errors.Is(err, os.ErrNotExist) {
		return file, nil
	}
	if err != nil {
		return cacheFile{}, err
	}
	if len(data) == 0 {
		return file, nil
	}
	if err := json.Unmarshal(data, &file); err != nil {
		return cacheFile{}, err
	}
	if file.Snapshots == nil {
		file.Snapshots = map[string]Snapshot{}
	}
	return file, nil
}
