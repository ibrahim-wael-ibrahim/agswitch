package quota

import (
	"context"

	"github.com/ibrahim-wael/agswitch/internal/credentials"
)

type Provider interface {
	Fetch(context.Context, credentials.Credential) (Snapshot, error)
}
