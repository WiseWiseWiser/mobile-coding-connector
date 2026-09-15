package commandcodeusage

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xhd2015/agent-pro/agent/commandcode"
)

func writeCommandCodeHome(t *testing.T, apiKey string) string {
	t.Helper()
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, "auth.json"), []byte(`{"apiKey":"`+apiKey+`","userName":"acct"}`), 0600); err != nil {
		t.Fatalf("write auth.json: %v", err)
	}
	return home
}

// fixtureAPIServer serves the four read-only endpoints the usage overlay reads,
// with numbers that reproduce the CLI's "5% used / 5h 15% / Weekly 11%" line.
func fixtureAPIServer(t *testing.T) *httptest.Server {
	t.Helper()
	// Anchor every timestamp to the real clock so the derived countdowns stay
	// valid whenever the suite runs.
	now := time.Now()
	periodStart := now.Add(-4 * 24 * time.Hour)
	periodEnd := now.Add(26*24*time.Hour - time.Hour)
	fiveHourReset := now.Add(2 * time.Hour).UnixMilli()
	weeklyReset := now.Add(2*24*time.Hour + 6*time.Hour).UnixMilli()

	bodies := map[string]string{
		"/alpha/whoami": `{"success":true,"user":{"id":"u1","name":"acct","email":"acct@example.com","userName":"acct"},"org":null}`,
		"/alpha/billing/credits": fmt.Sprintf(
			`{"credits":{"monthlyCredits":66.19,"purchasedCredits":0,"freeCredits":0},`+
				`"windowLimits":{"limited":true,`+
				`"fiveHour":{"used":2.1,"cap":14,"resetAt":%d},`+
				`"weekly":{"used":3.8,"cap":35,"resetAt":%d}}}`,
			fiveHourReset, weeklyReset),
		"/alpha/billing/subscriptions": fmt.Sprintf(
			`{"success":true,"data":{"id":"sub_1","status":"active","planId":"individual-goat","quantity":1,`+
				`"currentPeriodStart":%q,"currentPeriodEnd":%q}}`,
			periodStart.Format(time.RFC3339), periodEnd.Format(time.RFC3339)),
		"/alpha/usage/summary": `{"totalCount":1870,"totalCost":3.81,"completedCount":1870}`,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := bodies[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server
}

func TestFetchSnapshotFromFixtureAPI(t *testing.T) {
	server := fixtureAPIServer(t)
	home := writeCommandCodeHome(t, "test-key")
	svc := newService(home, server.URL, nil)

	resp := TestExported_FetchOnce(t, svc)
	if resp.Status != StatusReady {
		t.Fatalf("status = %q, error = %q", resp.Status, resp.Error)
	}
	if resp.PlanName != "GOAT" || resp.PlanStatus != "active" {
		t.Fatalf("plan = %q/%q, want GOAT/active", resp.PlanName, resp.PlanStatus)
	}
	if resp.CyclePercent != 5 {
		t.Fatalf("cycle percent = %d, want 5", resp.CyclePercent)
	}
	if resp.CycleRemaining != "$66.19" {
		t.Fatalf("cycle remaining = %q, want $66.19", resp.CycleRemaining)
	}
	if resp.CycleRequests != "1,870" {
		t.Fatalf("cycle requests = %q, want 1,870", resp.CycleRequests)
	}
	if resp.DaysToRenewal != 26 {
		t.Fatalf("days to renewal = %d, want 26", resp.DaysToRenewal)
	}
	if resp.FiveHourPercent != 15 || resp.WeeklyPercent != 11 {
		t.Fatalf("windows = %d/%d, want 15/11", resp.FiveHourPercent, resp.WeeklyPercent)
	}
	if resp.UsageURL != "https://commandcode.ai/acct/settings/usage" {
		t.Fatalf("usage url = %q", resp.UsageURL)
	}
	if !strings.Contains(resp.Detail, "GOAT Plan") || !strings.Contains(resp.Detail, "5% used") {
		t.Fatalf("detail missing plan headline:\n%s", resp.Detail)
	}
	if strings.Contains(resp.Detail, "\x1b[") {
		t.Fatalf("detail must be plain text:\n%s", resp.Detail)
	}

	if got := TitleSuffix(resp); got != "5%" {
		t.Fatalf("title suffix = %q, want 5%%", got)
	}
	want := "5% used, $66.19 left, 1,870 requests, 5h 15%, Weekly 11%, renews in 26d"
	if got := FormatBody(resp); got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestFetchFailsWithoutCredentials(t *testing.T) {
	server := fixtureAPIServer(t)
	home := t.TempDir() // no auth.json
	svc := newService(home, server.URL, nil)

	resp := TestExported_FetchOnce(t, svc)
	if resp.Status != StatusError {
		t.Fatalf("status = %q, want error", resp.Status)
	}
	if !strings.Contains(resp.Error, "not authenticated") {
		t.Fatalf("error = %q, want not authenticated", resp.Error)
	}
	if resp.Detail != "" || resp.UsageURL != "" {
		t.Fatalf("error response must not carry a panel: %+v", resp)
	}
	if resp.UpdatedAt == "" {
		t.Fatal("error response must stamp updated_at")
	}
}

func TestFetchFailsWhenAPIDown(t *testing.T) {
	server := fixtureAPIServer(t)
	home := writeCommandCodeHome(t, "test-key")
	url := server.URL
	server.Close()

	svc := newService(home, url, nil)
	resp := TestExported_FetchOnce(t, svc)
	if resp.Status != StatusError {
		t.Fatalf("status = %q, want error", resp.Status)
	}
	if resp.Error != commandcode.ErrNetwork {
		t.Fatalf("error = %q, want %q", resp.Error, commandcode.ErrNetwork)
	}
}

func TestGetRecomputesWindowCountdowns(t *testing.T) {
	home := writeCommandCodeHome(t, "test-key")
	svc := newService(home, "", nil)

	base := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	TestExported_SeedReady(svc, Snapshot{
		PlanName:        "GOAT",
		CyclePercent:    5,
		CycleRemaining:  "$66.19",
		DaysToRenewal:   26,
		FiveHourPercent: 15,
		WeeklyPercent:   11,
		FiveHourResetAt: base.Add(2 * time.Hour),
		WeeklyResetAt:   base.Add(2*24*time.Hour + 6*time.Hour),
	})
	TestExported_SetNow(svc, base)

	resp := svc.Get()
	if resp.FiveHourLeft != "left 2h" {
		t.Fatalf("five hour left = %q, want left 2h", resp.FiveHourLeft)
	}
	if resp.WeeklyLeft != "left 2d6h" {
		t.Fatalf("weekly left = %q, want left 2d6h", resp.WeeklyLeft)
	}

	TestExported_SetNow(svc, base.Add(time.Hour))
	resp = svc.Get()
	if resp.FiveHourLeft != "left 1h" {
		t.Fatalf("five hour left after an hour = %q, want left 1h", resp.FiveHourLeft)
	}
	if resp.WeeklyLeft != "left 2d5h" {
		t.Fatalf("weekly left after an hour = %q, want left 2d5h", resp.WeeklyLeft)
	}
}

func TestInjectedFetcherDrivesReadyAndErrorStates(t *testing.T) {
	home := writeCommandCodeHome(t, "test-key")

	ready := newService(home, "", nil)
	TestExported_SetFetcher(ready, func(context.Context) (*Snapshot, error) {
		return &Snapshot{PlanName: "Go", CyclePercent: 2, CycleRemaining: "$9.81", UsageURL: "https://commandcode.ai/x/settings/usage"}, nil
	})
	resp := TestExported_FetchOnce(t, ready)
	if resp.Status != StatusReady || resp.CyclePercent != 2 {
		t.Fatalf("ready resp = %+v", resp)
	}

	failing := newService(home, "", nil)
	TestExported_SetFetcher(failing, func(context.Context) (*Snapshot, error) {
		return nil, fmt.Errorf("mock fetch failed")
	})
	resp = TestExported_FetchOnce(t, failing)
	if resp.Status != StatusError || resp.Error != "mock fetch failed" {
		t.Fatalf("error resp = %+v", resp)
	}
}

func TestWindowPercent(t *testing.T) {
	if got := windowPercent(&commandcode.WindowSpan{Used: 3, Cap: 0}); got != unknownPercent {
		t.Fatalf("windowPercent with no cap = %d, want %d", got, unknownPercent)
	}
	if got := windowPercent(&commandcode.WindowSpan{Used: 2, Cap: 14}); got != 14 {
		t.Fatalf("windowPercent(2/14) = %d, want 14", got)
	}
	if got := windowPercent(&commandcode.WindowSpan{Used: 40, Cap: 35}); got != 100 {
		t.Fatalf("windowPercent clamps to 100, got %d", got)
	}
}
