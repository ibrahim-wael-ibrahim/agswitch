package switcher

import "time"

type Transaction struct {
	Profile   string
	StartedAt time.Time
	Steps     []string
}

func NewTransaction(p string) Transaction {
	return Transaction{Profile: p, StartedAt: time.Now().UTC()}
}
