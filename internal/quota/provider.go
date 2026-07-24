package quota

import "context"

type Provider interface {
	Fetch(ctx context.Context, profile string) (Snapshot, error)
}
