//go:build linux

package process

import "context"

type DBusQuitter struct{}

func (DBusQuitter) Quit(ctx context.Context, executable string) error {
	return nil
}
