package diskmon

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/xhd2015/dot-pkgs/go-pkgs/disk/reclaim"
)

// cleanupTestPlan fakes the engine scan: one step per requested id, 1 GiB each.
func cleanupTestPlan(ids []string) *reclaim.Plan {
	plan := &reclaim.Plan{ScannedAt: "2026-01-01T00:00:00Z", ScanMillis: 5}
	for _, id := range ids {
		size := int64(1 << 30)
		plan.Steps = append(plan.Steps, reclaim.PlanStep{ID: id, SizeBytes: &size})
	}
	return plan
}

func newCleanupTestService(t *testing.T) (*cleanupService, *[]string) {
	t.Helper()
	var scans []string
	svc := newCleanupService()
	svc.home = func() (string, error) { return "/Users/tester", nil }
	svc.scan = func(_ reclaim.Options, ids []string) (*reclaim.Plan, error) {
		scans = append(scans, strings.Join(ids, ","))
		return cleanupTestPlan(ids), nil
	}
	return svc, &scans
}

func getCleanupPlan(t *testing.T, svc *cleanupService) *CleanupPlan {
	t.Helper()
	mux := http.NewServeMux()
	(&cleanupAPI{service: svc}).register(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/server/disk/cleanup/plan", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("plan status = %d, body %s", rec.Code, rec.Body.String())
	}
	var plan CleanupPlan
	if err := json.Unmarshal(rec.Body.Bytes(), &plan); err != nil {
		t.Fatalf("decode plan: %v", err)
	}
	return &plan
}

// waitMeasured polls like the page does until the background pass lands.
func waitMeasured(t *testing.T, svc *cleanupService) *CleanupPlan {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		plan, err := svc.Plan(false)
		if err != nil {
			t.Fatalf("plan: %v", err)
		}
		if !plan.Pending {
			return plan
		}
		if time.Now().After(deadline) {
			t.Fatal("measurement never finished")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func findClass(t *testing.T, plan *CleanupPlan, id string) CleanupClass {
	t.Helper()
	for _, c := range plan.Classes {
		if c.ID == id {
			return c
		}
	}
	t.Fatalf("class %q missing from plan", id)
	return CleanupClass{}
}

func TestCleanupPlanCoversEveryRegistryClass(t *testing.T) {
	svc, _ := newCleanupTestService(t)
	plan := waitMeasured(t, svc)

	infos := reclaim.StepInfos("/Users/tester")
	if len(plan.Classes) != len(infos) {
		t.Fatalf("classes = %d, registry = %d", len(plan.Classes), len(infos))
	}
	if len(plan.Groups) != 4 {
		t.Fatalf("groups = %d, want 4", len(plan.Groups))
	}
	for _, c := range plan.Classes {
		if c.Label == "" {
			t.Errorf("class %q has no label", c.ID)
		}
		if len(c.Actions) == 0 {
			t.Errorf("class %q has no actions", c.ID)
		}
		if c.Command == "" {
			t.Errorf("class %q has no command", c.ID)
		}
		// Labels are host-localized: the engine's Chinese summaries must not
		// leak into the panel.
		for _, r := range c.Label {
			if r > 0x2E80 {
				t.Errorf("class %q label %q is not English", c.ID, c.Label)
			}
		}
	}
}

func TestCleanupPlanActionKinds(t *testing.T) {
	svc, _ := newCleanupTestService(t)
	plan := waitMeasured(t, svc)

	cases := []struct {
		id     string
		kind   string
		text   string
		okKind bool
	}{
		{id: "sandbox-gocache", kind: cleanupActionRemove, text: "~/.sandbox/gocache"},
		{id: "go-build-cache", kind: cleanupActionOfficial, text: "go clean -cache"},
		// go-mod-cache is inventory-only: the row names the official command but
		// the engine never runs it, so the badge must not claim "official".
		{id: "go-mod-cache", kind: cleanupActionNone, text: "go clean -modcache"},
		{id: "shipit", kind: cleanupActionNone, text: "—"},
		{id: "dirs-cleaner", kind: cleanupActionSudo, text: ""},
	}
	for _, tc := range cases {
		c := findClass(t, plan, tc.id)
		if c.Actions[0].Kind != tc.kind {
			t.Errorf("%s kind = %q, want %q", tc.id, c.Actions[0].Kind, tc.kind)
		}
		if tc.text != "" && c.Actions[0].Text != tc.text {
			t.Errorf("%s text = %q, want %q", tc.id, c.Actions[0].Text, tc.text)
		}
	}

	// Homebrew cleans with the official command first, then removes the cache
	// directory the official command leaves behind.
	brew := findClass(t, plan, "brew-cleanup")
	if len(brew.Actions) != 2 {
		t.Fatalf("brew actions = %d, want 2", len(brew.Actions))
	}
	if !strings.HasPrefix(brew.Actions[0].Text, "brew cleanup") {
		t.Errorf("brew action 0 = %q", brew.Actions[0].Text)
	}
	if brew.Actions[1].Kind != cleanupActionRemove || brew.Actions[1].Text != "~/Library/Caches/Homebrew" {
		t.Errorf("brew action 1 = %+v", brew.Actions[1])
	}
}

func TestCleanupPlanReadOnlyAndTotals(t *testing.T) {
	svc, _ := newCleanupTestService(t)
	plan := waitMeasured(t, svc)

	mod := findClass(t, plan, "go-mod-cache")
	if !mod.ReadOnly || mod.Selected {
		t.Errorf("go-mod-cache readOnly=%v selected=%v, want true/false", mod.ReadOnly, mod.Selected)
	}
	if mod.Group != cleanupGroupInventory {
		t.Errorf("go-mod-cache group = %q", mod.Group)
	}

	system := findClass(t, plan, "ios-sim-runtimes")
	if !system.NeedsAdmin || system.Selected {
		t.Errorf("ios-sim-runtimes needsAdmin=%v selected=%v, want true/false", system.NeedsAdmin, system.Selected)
	}

	// Every class was measured except the system tail, so the totals must add up
	// to the number of measured classes.
	if plan.Totals.SelectedBytes == 0 {
		t.Error("selectedBytes = 0, want the 8 default classes")
	}
	if plan.Totals.InventoryBytes == 0 {
		t.Error("inventoryBytes = 0, want shipit + go-mod-cache")
	}
	if plan.Totals.Unmeasured != 3 {
		t.Errorf("unmeasured = %d, want 3 system classes", plan.Totals.Unmeasured)
	}
}

func TestCleanupPlanServesSkeletonFirst(t *testing.T) {
	svc, _ := newCleanupTestService(t)
	plan := getCleanupPlan(t, svc)
	if !plan.Pending {
		t.Fatal("first paint must report pending while the pass runs")
	}
	if len(plan.Classes) != len(reclaim.StepInfos("/Users/tester")) {
		t.Fatalf("skeleton classes = %d, want the full registry", len(plan.Classes))
	}
	if class := findClass(t, plan, "sandbox-gocache"); class.Measured {
		t.Errorf("skeleton row already measured: %+v", class)
	}
	waitMeasured(t, svc)
}

func TestCleanupPlanCachesAndRefreshes(t *testing.T) {
	svc, scans := newCleanupTestService(t)
	waitMeasured(t, svc)
	if len(*scans) != 1 {
		t.Fatalf("scans = %d, want 1", len(*scans))
	}
	getCleanupPlan(t, svc)
	getCleanupPlan(t, svc)
	if len(*scans) != 1 {
		t.Fatalf("scans = %d, want 1 (fresh plan is served from cache)", len(*scans))
	}

	mux := http.NewServeMux()
	(&cleanupAPI{service: svc}).register(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/server/disk/cleanup/plan?refresh=1", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("refresh status = %d", rec.Code)
	}
	waitMeasured(t, svc)
	if len(*scans) != 2 {
		t.Fatalf("scans = %d, want 2 after refresh=1", len(*scans))
	}

	// The measurement pass must never include the system classes: they shell out
	// to xcrun and `log show` and would hang the panel.
	for _, ids := range *scans {
		if strings.Contains(ids, "ios-sim-runtimes") || strings.Contains(ids, "dirs-cleaner") {
			t.Errorf("plan scan requested a system class: %s", ids)
		}
	}
}

func TestCleanupPlanInvalidateDropsSizes(t *testing.T) {
	svc, _ := newCleanupTestService(t)
	waitMeasured(t, svc)
	svc.finish() // what a completed run does
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	(&cleanupAPI{service: svc}).register(mux)
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/server/disk/cleanup/run", nil))
	plan := getCleanupPlan(t, svc)
	if !plan.Pending {
		t.Error("plan after a run must re-measure")
	}
	if class := findClass(t, plan, "sandbox-gocache"); class.SizeBytes != nil {
		t.Errorf("stale size survived invalidation: %v", *class.SizeBytes)
	}
	svc.finish()
}

func TestCleanupRunSudoNeverSkipsAdminClasses(t *testing.T) {
	svc, _ := newCleanupTestService(t)
	var ran []string
	svc.run = func(opts reclaim.Options) (*reclaim.Result, error) {
		ran = opts.Only
		return &reclaim.Result{
			TotalFreed: 1 << 20,
			Steps:      []reclaim.StepResult{{ID: "sandbox-gocache", Freed: 1 << 20}},
		}, nil
	}

	body := strings.NewReader(`{"ids":["sandbox-gocache","ios-sim-runtimes","dirs-cleaner"],"sudo":"never"}`)
	mux := http.NewServeMux()
	(&cleanupAPI{service: svc}).register(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/server/disk/cleanup/run", body))

	if rec.Code != http.StatusOK {
		t.Fatalf("run status = %d, body %s", rec.Code, rec.Body.String())
	}
	if strings.Join(ran, ",") != "sandbox-gocache" {
		t.Fatalf("engine ran %v, want only the unprivileged class", ran)
	}
	out := rec.Body.String()
	if !strings.Contains(out, `"type":"done"`) {
		t.Fatalf("stream has no done frame:\n%s", out)
	}
	if !strings.Contains(out, `"status":"skipped"`) {
		t.Errorf("stream has no skipped frame:\n%s", out)
	}
	for _, id := range []string{"ios-sim-runtimes", "dirs-cleaner"} {
		if !strings.Contains(out, id) {
			t.Errorf("skipped frame missing %q", id)
		}
	}
	if !strings.Contains(out, "needs administrator authorization") {
		t.Errorf("skipped reason missing:\n%s", out)
	}
}

func TestCleanupRunAllClassesSkippedStillStreams(t *testing.T) {
	svc, _ := newCleanupTestService(t)
	svc.run = func(opts reclaim.Options) (*reclaim.Result, error) {
		t.Fatalf("engine ran %v, want no run when every class is skipped", opts.Only)
		return nil, nil
	}

	body := strings.NewReader(`{"ids":["ios-sim-runtimes","dirs-cleaner"],"sudo":"never"}`)
	mux := http.NewServeMux()
	(&cleanupAPI{service: svc}).register(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/server/disk/cleanup/run", body))

	if rec.Code != http.StatusOK {
		t.Fatalf("run status = %d, body %s", rec.Code, rec.Body.String())
	}
	out := rec.Body.String()
	for _, id := range []string{"ios-sim-runtimes", "dirs-cleaner"} {
		if !strings.Contains(out, id) {
			t.Errorf("skipped frame missing %q", id)
		}
	}
	if !strings.Contains(out, "needs administrator authorization") {
		t.Errorf("skipped reason missing:\n%s", out)
	}
	if !strings.Contains(out, `"type":"done"`) {
		t.Errorf("stream has no done frame:\n%s", out)
	}
}

func TestCleanupRunRejectsReadOnlyClass(t *testing.T) {
	svc, _ := newCleanupTestService(t)
	svc.run = func(reclaim.Options) (*reclaim.Result, error) {
		t.Fatal("engine must not run for a read-only class")
		return nil, nil
	}
	mux := http.NewServeMux()
	(&cleanupAPI{service: svc}).register(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/server/disk/cleanup/run",
		strings.NewReader(`{"ids":["go-mod-cache"],"sudo":"ask"}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "read-only") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestCleanupRunRejectsConcurrentRun(t *testing.T) {
	svc, _ := newCleanupTestService(t)
	if !svc.tryStart() {
		t.Fatal("tryStart on a fresh service = false")
	}
	defer svc.finish()
	mux := http.NewServeMux()
	(&cleanupAPI{service: svc}).register(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/server/disk/cleanup/run",
		strings.NewReader(`{"ids":["sandbox-gocache"],"sudo":"ask"}`)))
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
}

func TestCleanupMeasureRejectsUnknownAndReadOnly(t *testing.T) {
	svc, _ := newCleanupTestService(t)
	mux := http.NewServeMux()
	(&cleanupAPI{service: svc}).register(mux)

	for _, tc := range []struct{ body, want string }{
		{`{"ids":["nope"]}`, "unknown cleanup class"},
		{`{"ids":["go-mod-cache"]}`, "read-only"},
		{`{"ids":[]}`, "no cleanup class ids"},
	} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/server/disk/cleanup/measure", strings.NewReader(tc.body)))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", tc.body, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), tc.want) {
			t.Errorf("%s: body = %s", tc.body, rec.Body.String())
		}
	}
}

func TestCleanupMeasureCachesSystemSizes(t *testing.T) {
	svc, scans := newCleanupTestService(t)
	waitMeasured(t, svc)
	mux := http.NewServeMux()
	(&cleanupAPI{service: svc}).register(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/server/disk/cleanup/measure",
		strings.NewReader(`{"ids":["ios-sim-runtimes","ios-sim-runtimes"]}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	if len(*scans) != 2 || (*scans)[1] != "ios-sim-runtimes" {
		t.Fatalf("scans = %v, want one deduped ios-sim-runtimes probe", *scans)
	}

	plan := getCleanupPlan(t, svc)
	system := findClass(t, plan, "ios-sim-runtimes")
	if !system.Measured || system.SizeBytes == nil {
		t.Fatalf("system class not merged from the on-demand measurement: %+v", system)
	}
}

func TestCleanupMethodAndBodyValidation(t *testing.T) {
	svc, _ := newCleanupTestService(t)
	mux := http.NewServeMux()
	(&cleanupAPI{service: svc}).register(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/server/disk/cleanup/plan", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST plan = %d, want 405", rec.Code)
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/server/disk/cleanup/run",
		strings.NewReader(`{"ids":["sandbox-gocache"],"sudo":"yolo"}`)))
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "unknown sudo policy") {
		t.Errorf("bad sudo policy = %d %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/server/disk/cleanup/run",
		strings.NewReader(`{"ids":["sandbox-gocache"],"bogus":1}`)))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("unknown field = %d, want 400", rec.Code)
	}
}
