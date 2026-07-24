package quota

import (
	"context"
	"errors"
)

var ErrNotImplemented = errors.New("not implemented")

type GoogleInternalProvider struct{}

func (GoogleInternalProvider) Fetch(ctx context.Context, profile string) (Snapshot, error) {
	return Snapshot{}, ErrNotImplemented
}
