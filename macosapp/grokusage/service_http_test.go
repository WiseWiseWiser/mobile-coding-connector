package grokusage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultFetcher_FixtureEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fx.json")
	if err := os.WriteFile(path, []byte(`{"weekly_limit":"6%","next_reset":"July 9, 16:55 PT"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(envUsageFixture, path)

	svc := newService(defaultFetcher)
	out := svc.TestExported_FetchOnce(t)
	if out.Status != StatusReady {
		t.Fatalf("status=%s err=%q", out.Status, out.Error)
	}
	if out.WeeklyLimit != "6%" || out.NextReset != "July 9, 16:55 PT" {
		t.Fatalf("out=%+v", out)
	}
	if out.ResetAt == "" || out.TimeLeft == "" {
		t.Fatalf("structured fields empty: %+v", out)
	}
}

func TestDefaultFetcher_LiveHTTP(t *testing.T) {
	if testing.Short() {
		t.Skip("live network")
	}
	auth := filepath.Join(os.Getenv("HOME"), ".grok", "auth.json")
	if _, err := os.Stat(auth); err != nil {
		t.Skip("no ~/.grok/auth.json")
	}
	t.Setenv(envUsageFixture, "") // clear
	_ = os.Unsetenv(envUsageFixture)

	svc := newService(defaultFetcher)
	out := svc.TestExported_FetchOnce(t)
	if out.Status != StatusReady {
		t.Fatalf("status=%s err=%q", out.Status, out.Error)
	}
	if out.WeeklyLimit == "" {
		t.Fatalf("empty weekly: %+v", out)
	}
}
