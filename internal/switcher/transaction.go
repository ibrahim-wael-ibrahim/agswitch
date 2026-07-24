package switcher

import "time"

type Transaction struct {
	Profile   string
	StartedAt time.Time
	Steps     []string
}

func NewTransaction(profile string) Transaction {
	return Transaction{Profile: profile, StartedAt: time.Now().UTC()}
}
