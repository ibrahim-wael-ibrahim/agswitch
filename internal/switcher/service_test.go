package switcher

import (
	"context"
	"errors"
	"github.com/ibrahim-wael/agswitch/internal/credentials"
	"testing"
	"time"
)

type ma struct {
	c   credentials.Credential
	has bool
}

func (m *ma) Load(context.Context) (credentials.Credential, error) {
	if !m.has {
		return credentials.Credential{}, credentials.ErrNotFound
	}
	return m.c, nil
}
func (m *ma) Save(_ context.Context, c credentials.Credential) error {
	m.c = c
	m.has = true
	return nil
}
func (m *ma) Clear(context.Context) error { m.has = false; return nil }

type mp map[string]credentials.Credential

func (m mp) Load(_ context.Context, n string) (credentials.Credential, error) {
	v, ok := m[n]
	if !ok {
		return credentials.Credential{}, credentials.ErrNotFound
	}
	return v, nil
}

type fp struct{ running, fail bool }

func (p *fp) Stop(context.Context) error { p.running = false; return nil }
func (p *fp) Start(context.Context) error {
	if p.fail {
		return errors.New("start failed")
	}
	p.running = true
	return nil
}
func (p *fp) Running(context.Context) (bool, error) { return p.running, nil }
func TestRollback(t *testing.T) {
	old := credentials.New([]byte(`{"a":1}`))
	n := credentials.New([]byte(`{"a":2}`))
	a := &ma{old, true}
	p := &fp{running: true, fail: true}
	s := Service{ActiveStore: a, ProfileStore: mp{"new": n}, Process: p}
	if s.SwitchWithOptions(context.Background(), "new", Options{StartupTimeout: time.Millisecond}) == nil {
		t.Fatal("expected error")
	}
	if a.c.Fingerprint != old.Fingerprint {
		t.Fatal("not restored")
	}
}
