package account

import "context"

type Repository interface {
	List(ctx context.Context) ([]Account, error)
	Get(ctx context.Context, id string) (Account, error)
	Save(ctx context.Context, account Account) error
	Delete(ctx context.Context, id string) error
}
