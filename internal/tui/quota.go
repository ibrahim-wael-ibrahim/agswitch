package tui

import "github.com/ibrahim-wael/agswitch/internal/quota"

func (m *Model) SetQuota(snapshot quota.Snapshot) {
	m.Snapshot = snapshot
}
