package grokusage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	dotgrokusage "github.com/xhd2015/dot-pkgs/go-pkgs/shell/grok/usage"
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
	// This account is monthly-uncapped; preferred view is weekly credits ("N%").
	if out.WeeklyLimit[len(out.WeeklyLimit)-1] != '%' {
		t.Fatalf("want percent weekly_limit for uncapped monthly, got %+v", out)
	}
	if out.Period != "weekly" {
		t.Fatalf("want period=weekly for this account, got %+v", out)
	}
}

func TestMapBillingSnapshot_WeeklyCredits(t *testing.T) {
	snap := dotgrokusage.Snapshot{
		UsedPercent: 2,
		PeriodType:  dotgrokusage.PeriodWeekly,
		ResetAt:     time.Date(2026, 9, 4, 0, 55, 0, 0, time.UTC),
	}
	out := mapBillingSnapshot(snap)
	if out.WeeklyLimit != "2%" || out.Period != "weekly" {
		t.Fatalf("out = %+v", out)
	}
	if out.NextReset == "" {
		t.Fatal("want NextReset")
	}
}

func TestMapBillingSnapshot_UncappedNoInventPercent(t *testing.T) {
	snap := dotgrokusage.Snapshot{Used: 73, UsedPercent: -1, MonthlyLimit: 0}
	out := mapBillingSnapshot(snap)
	if out.WeeklyLimit != "73" {
		t.Fatalf("out = %+v", out)
	}
}

func TestMapBillingSnapshot_WeeklyNoPercentDefaultsZero(t *testing.T) {
	// Uncapped SuperGrok: the weekly credits period parsed but the backend
	// reports no percent — mirror grok.com and default the usage bar to 0%.
	snap := dotgrokusage.Snapshot{Used: 0, UsedPercent: -1, PeriodType: dotgrokusage.PeriodWeekly}
	out := mapBillingSnapshot(snap)
	if out.WeeklyLimit != "0%" || out.Period != "weekly" {
		t.Fatalf("out = %+v", out)
	}
}
