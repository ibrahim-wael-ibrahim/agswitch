package tui

import "github.com/ibrahim-wael/agswitch/internal/quota"

// SetQuota updates or inserts a snapshot for compatibility with callers that
// provide one profile at a time.
func (m *Model) SetQuota(snapshot quota.Snapshot) {
	for index := range m.Results {
		if m.Results[index].Profile == snapshot.Profile {
			m.Results[index].Snapshot = snapshot
			m.Results[index].Err = nil
			return
		}
	}
	m.Results = append(m.Results, quota.Result{Profile: snapshot.Profile, Snapshot: snapshot})
}
