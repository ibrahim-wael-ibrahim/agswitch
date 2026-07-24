package brand

import "testing"

func TestVersionLabel(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"":         "dev",
		"dev":      "dev",
		"v1.0.0":   "v1.0.0",
		"vv1.0.0":  "v1.0.0",
		"1.0.0":    "v1.0.0",
		" V1.2.3 ": "v1.2.3",
	}
	for input, expected := range cases {
		if actual := VersionLabel(input); actual != expected {
			t.Fatalf("VersionLabel(%q) = %q; want %q", input, actual, expected)
		}
	}
}

func TestBannerContainsProjectIdentity(t *testing.T) {
	t.Parallel()
	banner := Banner("v1.0.0")
	for _, expected := range []string{"agswitch v1.0.0", Author, Repository} {
		if !containsPlain(banner, expected) {
			t.Fatalf("banner does not contain %q", expected)
		}
	}
}

func containsPlain(value, expected string) bool {
	return len(value) >= len(expected) && (value == expected || contains(value, expected))
}

func contains(value, expected string) bool {
	for index := 0; index+len(expected) <= len(value); index++ {
		if value[index:index+len(expected)] == expected {
			return true
		}
	}
	return false
}
