package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestStripManagementCredentialsKeepsApplicationCredentials(t *testing.T) {
	previous := getCredentialsFile()
	path := filepath.Join(t.TempDir(), "credentials")
	if err := os.WriteFile(path, []byte("fixture-management-token\n"), 0600); err != nil {
		t.Fatal(err)
	}
	SetCredentialsFile(path)
	defer SetCredentialsFile(previous)
	for _, token := range []string{"fixture-management-token", "application-token"} {
		r := httptest.NewRequest("GET", "http://public.example/", nil)
		r.Header.Set("Authorization", "Bearer "+token)
		r.AddCookie(&http.Cookie{Name: CookieName(), Value: "must-not-leak"})
		r.AddCookie(&http.Cookie{Name: "application-session", Value: "keep"})
		StripManagementCredentials(r)
		if _, err := r.Cookie(CookieName()); err == nil {
			t.Fatal("management cookie leaked")
		}
		if _, err := r.Cookie("application-session"); err != nil {
			t.Fatal("application cookie removed")
		}
		want := "Bearer application-token"
		if token == "fixture-management-token" {
			want = ""
		}
		if r.Header.Get("Authorization") != want {
			t.Fatalf("unexpected Authorization %q", r.Header.Get("Authorization"))
		}
	}
}
