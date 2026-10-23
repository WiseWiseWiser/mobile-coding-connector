package config

import (
	"os"
	"path/filepath"
	"testing"
)

func useTempPortMappingAuthFile(t *testing.T) {
	t.Helper()
	oldFile := PortMappingAuthFile
	oldDataDir := DataDir
	dir := t.TempDir()
	DataDir = dir
	PortMappingAuthFile = filepath.Join(dir, "port-mapping-auth.json")
	t.Cleanup(func() {
		PortMappingAuthFile = oldFile
		DataDir = oldDataDir
	})
}

func TestPortMappingAuthRoundTrip(t *testing.T) {
	useTempPortMappingAuthFile(t)

	if got, err := GetPortMappingAuth(8422); err != nil || got != nil {
		t.Fatalf("GetPortMappingAuth empty = %#v, %v", got, err)
	}

	err := SetPortMappingAuth(8422, PortAuth{
		Required:  true,
		User:      " alice ",
		TokenMode: PortAuthCustom,
		Token:     " s3cret ",
	})
	if err != nil {
		t.Fatalf("SetPortMappingAuth: %v", err)
	}
	got, err := GetPortMappingAuth(8422)
	if err != nil {
		t.Fatalf("GetPortMappingAuth: %v", err)
	}
	if got == nil || !got.Required || got.User != "alice" || got.TokenMode != PortAuthCustom || got.Token != "s3cret" || got.Ephemeral {
		t.Fatalf("saved = %#v", got)
	}

	if err := SetPortMappingAuth(8422, PortAuth{}); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if got, err := GetPortMappingAuth(8422); err != nil || got != nil {
		t.Fatalf("cleared = %#v, %v", got, err)
	}
}

func TestPortMappingAuthReadsLegacyFlat(t *testing.T) {
	useTempPortMappingAuthFile(t)
	raw := []byte(`{"8422":{"requireAuth":true,"authTokenMode":"shared"}}` + "\n")
	if err := os.WriteFile(PortMappingAuthFile, raw, 0600); err != nil {
		t.Fatal(err)
	}
	got, err := GetPortMappingAuth(8422)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || !got.Required || got.TokenMode != PortAuthShared {
		t.Fatalf("legacy = %#v", got)
	}
}

func TestGenerateEphemeralToken(t *testing.T) {
	a, err := GenerateEphemeralToken()
	if err != nil {
		t.Fatal(err)
	}
	b, err := GenerateEphemeralToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 32 || len(b) != 32 || a == b {
		t.Fatalf("tokens = %q %q", a, b)
	}
}

func TestDeletePortMappingAuthMissingIsOK(t *testing.T) {
	useTempPortMappingAuthFile(t)
	if err := DeletePortMappingAuth(9); err != nil {
		t.Fatalf("DeletePortMappingAuth: %v", err)
	}
}

func TestPortAuthPublicStripsToken(t *testing.T) {
	a := PortAuth{Required: true, Ephemeral: true, TokenMode: PortAuthEphemeral, Token: "secret"}
	pub := a.Public()
	if pub == nil || pub.Token != "" || !pub.Required || !pub.Ephemeral {
		t.Fatalf("Public = %#v", pub)
	}
	if a.Token != "secret" {
		t.Fatal("Public mutated the original")
	}
}
