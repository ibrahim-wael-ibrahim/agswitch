package switcher

import "context"

type Rollbacker interface {
	Rollback(ctx context.Context) error
}

type NoopRollbacker struct{}

func (NoopRollbacker) Rollback(ctx context.Context) error {
	return nil
}
