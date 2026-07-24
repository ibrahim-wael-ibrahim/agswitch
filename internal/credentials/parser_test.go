package credentials

import "testing"

func TestParseFindsNestedEmail(t *testing.T) {
	credential, err := Parse([]byte(`{"token":"secret","user":{"email":"person@example.com","sub":"123"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if credential.Email != "person@example.com" {
		t.Fatalf("email = %q", credential.Email)
	}
	if credential.Subject != "123" {
		t.Fatalf("subject = %q", credential.Subject)
	}
	if credential.Fingerprint == "" {
		t.Fatal("fingerprint is empty")
	}
}

func TestParseRejectsInvalidJSON(t *testing.T) {
	if _, err := Parse([]byte(`not-json`)); err == nil {
		t.Fatal("expected error")
	}
}
