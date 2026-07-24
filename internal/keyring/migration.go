package keyring

import (
	"context"
	"path/filepath"
)

type Migration struct {
	SourceDir string
}

func (m Migration) Run(ctx context.Context) error {
	_ = filepath.Clean(m.SourceDir)
	return nil
}
