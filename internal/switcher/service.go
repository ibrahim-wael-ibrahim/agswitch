package switcher

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ibrahim-wael/agswitch/internal/credentials"
)

type ActiveStore interface {
	Load(context.Context) (credentials.Credential, error)
	Save(context.Context, credentials.Credential) error
	Clear(context.Context) error
}
type ProfileStore interface {
	Load(context.Context, string) (credentials.Credential, error)
}
type ProcessManager interface {
	Stop(context.Context) error
	Start(context.Context) error
	Running(context.Context) (bool, error)
	ReloadBackend(context.Context) error
}
type StateStore interface {
	Commit(context.Context, string) error
}
type Locker interface {
	Lock(context.Context) (func() error, error)
}
type LaunchMode int

const (
	PreserveLaunchState LaunchMode = iota
	AlwaysLaunch
	NeverLaunch
)

type Options struct {
	LaunchMode     LaunchMode
	StartupTimeout time.Duration
	HotReload      bool
}
type Service struct {
	ActiveStore  ActiveStore
	ProfileStore ProfileStore
	Process      ProcessManager
	State        StateStore
	Locker       Locker
}

func (s Service) Switch(ctx context.Context, p string) error {
	return s.SwitchWithOptions(ctx, p, Options{})
}
func (s Service) SwitchWithOptions(ctx context.Context, p string, o Options) (err error) {
	if s.ActiveStore == nil || s.ProfileStore == nil || s.Process == nil {
		return errors.New("switcher dependencies are not configured")
	}
	if s.Locker != nil {
		u, e := s.Locker.Lock(ctx)
		if e != nil {
			return e
		}
		defer func() {
			if x := u(); err == nil && x != nil {
				err = x
			}
		}()
	}
	sel, e := s.ProfileStore.Load(ctx, p)
	if e != nil {
		return fmt.Errorf("load profile %q: %w", p, e)
	}
	prev, pe := s.ActiveStore.Load(ctx)
	had := pe == nil
	if pe != nil && !errors.Is(pe, credentials.ErrNotFound) {
		return pe
	}
	was, e := s.Process.Running(ctx)
	if e != nil {
		return e
	}
	if o.HotReload && !was {
		return errors.New("hot reload requires Antigravity to be running")
	}
	start := was
	if o.LaunchMode == AlwaysLaunch {
		start = true
	} else if o.LaunchMode == NeverLaunch {
		start = false
	} else if o.LaunchMode != PreserveLaunchState {
		return errors.New("invalid launch mode")
	}
	changed, stopped, committed := false, false, false
	defer func() {
		if err != nil && !committed {
			if r := s.rollback(ctx, prev, had, was, changed, stopped, o.HotReload); r != nil {
				err = errors.Join(err, r)
			}
		}
	}()
	if was && !o.HotReload {
		if err = s.Process.Stop(ctx); err != nil {
			return err
		}
		stopped = true
	}
	if err = s.ActiveStore.Save(ctx, sel); err != nil {
		return err
	}
	changed = true
	v, e := s.ActiveStore.Load(ctx)
	if e != nil || v.Fingerprint != sel.Fingerprint {
		return errors.New("active credential verification failed")
	}
	if o.HotReload {
		if err = s.Process.ReloadBackend(ctx); err != nil {
			return fmt.Errorf("reload Antigravity language server: %w", err)
		}
	} else if start {
		if err = s.Process.Start(ctx); err != nil {
			return err
		}
		t := o.StartupTimeout
		if t <= 0 {
			t = 10 * time.Second
		}
		if err = waitForProcess(ctx, s.Process, true, t); err != nil {
			return err
		}
	}
	if s.State != nil {
		if err = s.State.Commit(ctx, p); err != nil {
			return err
		}
	}
	committed = true
	return nil
}
func (s Service) rollback(ctx context.Context, p credentials.Credential, had, was, changed, stopped, hotReload bool) error {
	var out error
	if changed {
		if had {
			out = errors.Join(out, s.ActiveStore.Save(ctx, p))
		} else {
			out = errors.Join(out, s.ActiveStore.Clear(ctx))
		}
	}
	if hotReload && was && changed {
		out = errors.Join(out, s.Process.ReloadBackend(ctx))
		return out
	}
	running, e := s.Process.Running(ctx)
	if e != nil {
		return errors.Join(out, e)
	}
	if was && stopped && !running {
		out = errors.Join(out, s.Process.Start(ctx))
	}
	if !was && running {
		out = errors.Join(out, s.Process.Stop(ctx))
	}
	return out
}
func waitForProcess(ctx context.Context, m ProcessManager, expected bool, timeout time.Duration) error {
	d := time.NewTimer(timeout)
	defer d.Stop()
	t := time.NewTicker(200 * time.Millisecond)
	defer t.Stop()
	for {
		r, e := m.Running(ctx)
		if e != nil {
			return e
		}
		if r == expected {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-d.C:
			return errors.New("process state timeout")
		case <-t.C:
		}
	}
}
