package profile

import (
	"errors"
	"regexp"
)

var validName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

func Validate(name string) error {
	if !validName.MatchString(name) {
		return errors.New("profile name must be 1-64 characters and contain only letters, numbers, dot, underscore, or dash")
	}
	return nil
}
