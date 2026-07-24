package account

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestFileRepositoryLifecycle(t *testing.T) {
	repo := NewFileRepository(filepath.Join(t.TempDir(), "accounts.json"))
	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	repo.Now = func() time.Time { return now }

	item := Account{ID: "work", Email: "person@example.com", CredentialFingerprint: "abc"}
	if err := repo.Save(context.Background(), item); err != nil {
		t.Fatal(err)
	}
	loaded, err := repo.Get(context.Background(), "work")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.CreatedAt != now || loaded.UpdatedAt != now {
		t.Fatalf("unexpected timestamps: %#v", loaded)
	}
	if err := repo.Delete(context.Background(), "work"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Get(context.Background(), "work"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v", err)
	}
}
