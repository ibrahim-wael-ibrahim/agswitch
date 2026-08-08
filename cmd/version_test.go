package cmd

import "testing"

func TestResolvedVersionUsesExplicitBuildVersion(t *testing.T) {
	original := version
	defer func() { version = original }()
	version = "v1.2.3"
	if got := resolvedVersion(); got != "v1.2.3" {
		t.Fatalf("resolvedVersion() = %q; want v1.2.3", got)
	}
}

func TestResolvedVersionDoesNotDefaultToV100(t *testing.T) {
	original := version
	defer func() { version = original }()
	version = "dev"
	if got := resolvedVersion(); got == "v1.0.0" {
		t.Fatalf("resolvedVersion() unexpectedly returned fixed legacy version %q", got)
	}
}
