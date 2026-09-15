# Command Code Usage Doctests

Tests for `macosapp/commandcodeusage` — the in-process fetch, cache, and
rendering of one Command Code account (`--home ~/.sandbox/commandcode-v1/.commandcode`).

# DSN (Domain Specific Notion)

**Participants**

- **Command Code home** — a provider config directory holding `auth.json`
  (API key + account name). Every registered Command Code item names one.
- **Command Code API** — read-only overlay endpoints: `/alpha/whoami`,
  `/alpha/billing/credits`, `/alpha/billing/subscriptions`, `/alpha/usage/summary`.
- **commandcodeusage.Service** — fetches once, caches a `Response`, refreshes
  every 10 minutes. `Get()` recomputes the 5-hour and weekly countdowns from the
  cached reset instants, so time keeps moving between fetches.
- **Response** — plan name/status, cycle credit percent, cycle money remaining,
  request count, days to renewal, the two window percents, the usage URL, and
  the plain-text provider panel (`Detail`).
- **TitleSuffix** — the menu-bar percent: `5%` for a ready response, empty otherwise.
- **FormatBody** — the dropdown body without the label prefix:
  `5% used, $66.19 left, 1,870 requests, 5h 15%, Weekly 11%, renews in 26d`.
- **Fixture HTTP API** — the leaves serve the four endpoints with numbers that
  reproduce a real sandbox account; no live network.

**Behaviors**

- A ready fetch reports the GOAT plan as `active`, 5% of the cycle used,
  `$66.19` left, 1,870 requests, 26 days to renewal, and 15% / 11% windows.
- A home without `auth.json` fails with `not authenticated`, and the error
  response carries no panel and no usage URL.
- An unreachable API fails with the provider's network error.
- An injected fetch error becomes status `error` with the error message.
- `Get()` shortens the window countdowns as the clock advances, without refetching.
- The error response still stamps `updated_at`, so the menu bar can show staleness.

## Version

0.1.0

## Decision Tree

```
[command code usage]
 |
 +-- fetch/                                 (GROUP)  service fetch outcomes
 |    +-- ready-from-fixture-api/           (LEAF)   plan, cycle, windows, url
 |    +-- fails-without-credentials/        (LEAF)   no auth.json
 |    +-- fails-when-api-down/              (LEAF)   network error
 |    +-- fails-when-fetch-errors/          (LEAF)   injected error
 |
 +-- get/                                   (GROUP)  cache reads
 |    +-- recomputes-window-countdowns/     (LEAF)   countdown advances, no refetch
 |
 +-- render/                                (GROUP)  menu-bar text
      +-- title-and-body/                   (LEAF)   "5%" and the dropdown body
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `fetch/ready-from-fixture-api` | Fixture API → ready response with plan, cycle, windows, url |
| 2 | `fetch/fails-without-credentials` | No `auth.json` → error, no panel |
| 3 | `fetch/fails-when-api-down` | Closed API → provider network error |
| 4 | `fetch/fails-when-fetch-errors` | Injected error → status error with message |
| 5 | `get/recomputes-window-countdowns` | Second `Get()` at +1h shortens both countdowns |
| 6 | `render/title-and-body` | `TitleSuffix` → `5%`; `FormatBody` → full dropdown body |

## Parameter Coverage

| Leaf | Op | Home | API | Expect |
|------|-----|------|-----|--------|
| ready-from-fixture-api | ready | auth.json | fixture | ready, GOAT |
| fails-without-credentials | no-credentials | empty | fixture | error |
| fails-when-api-down | api-down | auth.json | closed | network error |
| fails-when-fetch-errors | injected-error | auth.json | — | injected message |
| recomputes-window-countdowns | countdown | auth.json | — | 2h→1h, 2d6h→2d5h |
| title-and-body | ready | auth.json | fixture | `5%`, full body |

## How to Run

```sh
doctest vet ./tests/commandcode-usage
doctest test ./tests/commandcode-usage/...
```

```go
import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/macosapp/commandcodeusage"
	"github.com/xhd2015/doctest/session"
)

type Request struct {
	Op string

	// Op=countdown: fixed clock instants (RFC3339) for the two Get calls.
	NowRFC3339  string
	ThenRFC3339 string
}

type Response struct {
	Status          string
	Error           string
	PlanName        string
	PlanStatus      string
	CyclePercent    int
	CycleRemaining  string
	CycleRequests   string
	DaysToRenewal   int
	FiveHourPercent int
	WeeklyPercent   int
	UsageURL        string
	Detail          string
	UpdatedAt       string

	// Rendered menu-bar text.
	Title string
	Body  string

	// Op=countdown
	FiveHourLeft      string
	WeeklyLeft        string
	FiveHourLeftAfter string
	WeeklyLeftAfter   string
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	resp := &Response{}
	home := writeCommandCodeHome(t, "test-key")

	switch req.Op {
	case "ready", "render":
		server := fixtureAPIServer(t)
		svc := commandcodeusage.TestExported_NewLiveService(home, server.URL)
		record(resp, commandcodeusage.TestExported_FetchOnce(t, svc))
		return resp, nil

	case "no-credentials":
		server := fixtureAPIServer(t)
		svc := commandcodeusage.TestExported_NewLiveService(t.TempDir(), server.URL)
		record(resp, commandcodeusage.TestExported_FetchOnce(t, svc))
		return resp, nil

	case "api-down":
		server := fixtureAPIServer(t)
		url := server.URL
		server.Close()
		svc := commandcodeusage.TestExported_NewLiveService(home, url)
		record(resp, commandcodeusage.TestExported_FetchOnce(t, svc))
		return resp, nil

	case "injected-error":
		svc := commandcodeusage.TestExported_NewService(home, "")
		commandcodeusage.TestExported_SetFetcher(svc, func(context.Context) (*commandcodeusage.Snapshot, error) {
			return nil, fmt.Errorf("token rejected by provider")
		})
		record(resp, commandcodeusage.TestExported_FetchOnce(t, svc))
		return resp, nil

	case "countdown":
		base, err := time.Parse(time.RFC3339, req.NowRFC3339)
		if err != nil {
			return nil, err
		}
		svc := commandcodeusage.TestExported_NewService(home, "")
		commandcodeusage.TestExported_SeedReady(svc, commandcodeusage.Snapshot{
			PlanName:        "GOAT",
			CyclePercent:    5,
			CycleRemaining:  "$66.19",
			DaysToRenewal:   26,
			FiveHourPercent: 15,
			WeeklyPercent:   11,
			FiveHourResetAt: base.Add(2 * time.Hour),
			WeeklyResetAt:   base.Add(2*24*time.Hour + 6*time.Hour),
		})
		commandcodeusage.TestExported_SetNow(svc, base)
		first := svc.Get()
		resp.FiveHourLeft = first.FiveHourLeft
		resp.WeeklyLeft = first.WeeklyLeft

		later, err := time.Parse(time.RFC3339, req.ThenRFC3339)
		if err != nil {
			return nil, err
		}
		commandcodeusage.TestExported_SetNow(svc, later)
		second := svc.Get()
		resp.FiveHourLeftAfter = second.FiveHourLeft
		resp.WeeklyLeftAfter = second.WeeklyLeft
		resp.UpdatedAt = second.UpdatedAt
		return resp, nil

	default:
		return nil, fmt.Errorf("unknown op %q", req.Op)
	}
}

// record copies a service response into the doctest response.
func record(resp *Response, got commandcodeusage.Response) {
	resp.Status = string(got.Status)
	resp.Error = got.Error
	resp.PlanName = got.PlanName
	resp.PlanStatus = got.PlanStatus
	resp.CyclePercent = got.CyclePercent
	resp.CycleRemaining = got.CycleRemaining
	resp.CycleRequests = got.CycleRequests
	resp.DaysToRenewal = got.DaysToRenewal
	resp.FiveHourPercent = got.FiveHourPercent
	resp.WeeklyPercent = got.WeeklyPercent
	resp.UsageURL = got.UsageURL
	resp.Detail = got.Detail
	resp.UpdatedAt = got.UpdatedAt
	resp.Title = commandcodeusage.TitleSuffix(got)
	resp.Body = commandcodeusage.FormatBody(got)
}

// writeCommandCodeHome creates a provider config directory with credentials.
func writeCommandCodeHome(t *testing.T, apiKey string) string {
	t.Helper()
	home := t.TempDir()
	auth := `{"apiKey":"` + apiKey + `","userName":"acct"}`
	if err := os.WriteFile(filepath.Join(home, "auth.json"), []byte(auth), 0600); err != nil {
		t.Fatalf("write auth.json: %v", err)
	}
	return home
}

// fixtureAPIServer serves the four endpoints the usage overlay reads. Every
// timestamp is anchored to the real clock so countdowns stay valid.
func fixtureAPIServer(t *testing.T) *httptest.Server {
	t.Helper()
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
```
