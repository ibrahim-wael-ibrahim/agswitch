package profile

import "testing"

func TestValidate(t *testing.T) {
	for _, name := range []string{"work", "ibrahim-wael", "a.b_1"} {
		if err := Validate(name); err != nil {
			t.Fatalf("%q: %v", name, err)
		}
	}
	for _, name := range []string{"", "../secret", "white space", "-starts-with-dash"} {
		if err := Validate(name); err == nil {
			t.Fatalf("expected %q to be invalid", name)
		}
	}
}
