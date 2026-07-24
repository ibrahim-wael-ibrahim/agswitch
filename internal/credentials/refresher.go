package credentials

import "context"

type Refresher interface {
	Refresh(ctx context.Context, credential Credential) (Credential, error)
}

type StaticRefresher struct{}

func (StaticRefresher) Refresh(ctx context.Context, credential Credential) (Credential, error) {
	return credential, nil
}
