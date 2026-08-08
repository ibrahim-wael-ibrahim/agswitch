package quota

import (
	"strings"
	"testing"
)

func TestParseOAuthFailure(t *testing.T) {
	body := []byte("{\"error\":\"invalid_grant\",\"error_description\":\"Token has been expired or revoked.\",\"error_subtype\":\"invalid_rapt\"}")
	failure := parseOAuthFailure(400, body)
	if failure.Status != 400 {
		t.Fatalf("status = %d", failure.Status)
	}
	if failure.Code != "invalid_grant" {
		t.Fatalf("code = %q", failure.Code)
	}
	if failure.Description != "Token has been expired or revoked." {
		t.Fatalf("description = %q", failure.Description)
	}
	if failure.Subtype != "invalid_rapt" {
		t.Fatalf("subtype = %q", failure.Subtype)
	}
}

func TestParseOAuthFailureDoesNotEchoUnknownBody(t *testing.T) {
	secret := "super-secret-refresh-token"
	failure := parseOAuthFailure(401, []byte(secret))
	if strings.Contains(failure.Error(), secret) {
		t.Fatalf("diagnostic error leaked response body: %q", failure.Error())
	}
	if failure.Code != "" || failure.Description != "" || failure.Subtype != "" {
		t.Fatalf("unexpected parsed fields: %+v", failure)
	}
}

func TestCompareClientID(t *testing.T) {
	got := compareClientID("client.apps.googleusercontent.com", OAuthTokenInfo{IssuedTo: "client.apps.googleusercontent.com"})
	if got == nil || !*got {
		t.Fatalf("expected matching client id, got %v", got)
	}
	got = compareClientID("other.apps.googleusercontent.com", OAuthTokenInfo{Audience: "client.apps.googleusercontent.com"})
	if got == nil || *got {
		t.Fatalf("expected mismatching client id, got %v", got)
	}
	if compareClientID("", OAuthTokenInfo{IssuedTo: "client.apps.googleusercontent.com"}) != nil {
		t.Fatal("missing configured client id should be inconclusive")
	}
}

func TestSanitizeOAuthField(t *testing.T) {
	input := " invalid_client\n\t credentials rejected "
	got := sanitizeOAuthField(input)
	if got != "invalid_client credentials rejected" {
		t.Fatalf("sanitizeOAuthField() = %q", got)
	}
}
