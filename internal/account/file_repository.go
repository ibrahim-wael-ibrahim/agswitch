package account

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sort"
	"time"

	"github.com/ibrahim-wael/agswitch/internal/fsutil"
	"github.com/ibrahim-wael/agswitch/internal/profile"
)

type FileRepository struct {
	Path string
	Now  func() time.Time
}

type accountFile struct {
	Version  int       `json:"version"`
	Accounts []Account `json:"accounts"`
}

func NewFileRepository(path string) *FileRepository {
	return &FileRepository{Path: path, Now: time.Now}
}

func (r *FileRepository) List(ctx context.Context) ([]Account, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	file, err := r.read()
	if err != nil {
		return nil, err
	}
	accounts := append([]Account(nil), file.Accounts...)
	sort.Slice(accounts, func(i, j int) bool { return accounts[i].ID < accounts[j].ID })
	return accounts, nil
}

func (r *FileRepository) Get(ctx context.Context, id string) (Account, error) {
	accounts, err := r.List(ctx)
	if err != nil {
		return Account{}, err
	}
	for _, item := range accounts {
		if item.ID == id {
			return item, nil
		}
	}
	return Account{}, ErrNotFound
}

func (r *FileRepository) Save(ctx context.Context, item Account) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := profile.Validate(item.ID); err != nil {
		return err
	}
	file, err := r.read()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if r.Now != nil {
		now = r.Now().UTC()
	}
	found := false
	for index := range file.Accounts {
		if file.Accounts[index].ID != item.ID {
			continue
		}
		if item.CreatedAt.IsZero() {
			item.CreatedAt = file.Accounts[index].CreatedAt
		}
		item.UpdatedAt = now
		file.Accounts[index] = item
		found = true
		break
	}
	if !found {
		if item.CreatedAt.IsZero() {
			item.CreatedAt = now
		}
		item.UpdatedAt = now
		file.Accounts = append(file.Accounts, item)
	}
	return r.write(file)
}

func (r *FileRepository) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	file, err := r.read()
	if err != nil {
		return err
	}
	filtered := make([]Account, 0, len(file.Accounts))
	found := false
	for _, item := range file.Accounts {
		if item.ID == id {
			found = true
			continue
		}
		filtered = append(filtered, item)
	}
	if !found {
		return ErrNotFound
	}
	file.Accounts = filtered
	return r.write(file)
}

func (r *FileRepository) read() (accountFile, error) {
	file := accountFile{Version: 1, Accounts: []Account{}}
	data, err := os.ReadFile(r.Path)
	if errors.Is(err, os.ErrNotExist) {
		return file, nil
	}
	if err != nil {
		return accountFile{}, err
	}
	if len(data) == 0 {
		return accountFile{}, errors.New("accounts metadata file is empty")
	}
	if err := json.Unmarshal(data, &file); err != nil {
		return accountFile{}, err
	}
	if file.Version == 0 {
		file.Version = 1
	}
	seen := make(map[string]struct{}, len(file.Accounts))
	for _, item := range file.Accounts {
		if err := profile.Validate(item.ID); err != nil {
			return accountFile{}, err
		}
		if _, exists := seen[item.ID]; exists {
			return accountFile{}, errors.New("duplicate account id in metadata")
		}
		seen[item.ID] = struct{}{}
	}
	return file, nil
}

func (r *FileRepository) write(file accountFile) error {
	file.Version = 1
	sort.Slice(file.Accounts, func(i, j int) bool { return file.Accounts[i].ID < file.Accounts[j].ID })
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return fsutil.WriteFileAtomic(r.Path, data, 0o600)
}
