package app

import (
	"context"
	"testing"

	"github.com/ibrahim-wael/agswitch/internal/account"
	"github.com/ibrahim-wael/agswitch/internal/credentials"
)

func TestCloneAndRename(t *testing.T) {
	ctx := context.Background()
	credential := credentials.New([]byte(`{"email":"person@example.com","access_token":"token"}`))
	profiles := profilesMemory{"work": credential}
	accounts := accountsMemory{"work": {ID: "work", Email: "person@example.com", CredentialFingerprint: credential.Fingerprint}}
	service := Service{Profiles: profiles, Accounts: accounts}

	if err := service.Clone(ctx, "work", "backup", false); err != nil {
		t.Fatal(err)
	}
	if profiles["backup"].Fingerprint != credential.Fingerprint {
		t.Fatal("cloned credential fingerprint does not match")
	}
	if accounts["backup"].ID != "backup" {
		t.Fatalf("unexpected cloned metadata: %#v", accounts["backup"])
	}

	if err := service.Rename(ctx, "backup", "archive", false); err != nil {
		t.Fatal(err)
	}
	if _, ok := profiles["backup"]; ok {
		t.Fatal("source credential still exists after rename")
	}
	if _, ok := accounts["backup"]; ok {
		t.Fatal("source metadata still exists after rename")
	}
	if profiles["archive"].Fingerprint != credential.Fingerprint {
		t.Fatal("renamed credential fingerprint does not match")
	}
	if accounts["archive"].ID != "archive" {
		t.Fatalf("unexpected renamed metadata: %#v", accounts["archive"])
	}
}

func TestInfoDoesNotExposeCredential(t *testing.T) {
	ctx := context.Background()
	credential, err := credentials.Parse([]byte(`{"email":"person@example.com","access_token":"secret","credential_type":"oauth"}`))
	if err != nil {
		t.Fatal(err)
	}
	service := Service{
		Profiles: profilesMemory{"work": credential},
		Accounts: accountsMemory{"work": account.Account{ID: "work", Email: "person@example.com", CredentialFingerprint: credential.Fingerprint}},
	}
	info, err := service.Info(ctx, "work")
	if err != nil {
		t.Fatal(err)
	}
	if info.Storage != "system-keyring" || info.CredentialType != "oauth" {
		t.Fatalf("unexpected profile info: %#v", info)
	}
}
