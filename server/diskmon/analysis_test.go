package diskmon

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// A macOS walk starts at /System/Volumes/Data, so every path carries that
// prefix. Rules are written against "/" and the home directory and must still
// match, or the whole scan lands in "other".
func TestClassifierStripsDataVolume(t *testing.T) {
	home := "/Users/me"
	classifier := newClassifier(home, dataMount)

	cases := []struct {
		path        string
		category    string
		reclaimable bool
	}{
		{dataMount + "/Users/me/Library/Developer/Xcode/DerivedData/App-abc", "xcode", true},
		{dataMount + "/Users/me/Library/Caches/Homebrew/downloads", "dev-cache", true},
		{dataMount + "/Users/me/Downloads/installer.dmg", "downloads", false},
		{dataMount + "/Applications/Xcode.app/Contents", "other", false},
	}
	for _, testCase := range cases {
		category, reclaimable := classifier.classify(testCase.path)
		if category != testCase.category || reclaimable != testCase.reclaimable {
			t.Errorf("classify(%q) = (%s, %v), want (%s, %v)",
				testCase.path, category, reclaimable, testCase.category, testCase.reclaimable)
		}
	}

	if !classifier.isRuleRoot(dataMount + "/Users/me/Library/Developer") {
		t.Error("a category root under the data volume is not recognized")
	}
	if got, want := shortenHome(classifier.logical(dataMount+"/Users/me/Downloads"), home), "~/Downloads"; got != want {
		t.Errorf("logical path = %q, want %q", got, want)
	}
	if got := classifier.logical(dataMount); got != dataMount {
		t.Errorf("logical(volume root) = %q, want it unchanged", got)
	}
}

// The same walk, end to end: the fake volume prefix must not leak into the
// category totals or the entry paths shown on the page.
func TestScanStripsVolumePrefix(t *testing.T) {
	volume := t.TempDir()
	home := filepath.Join(volume, "Users", "me")
	derived := filepath.Join(home, "Library", "Developer", "Xcode", "DerivedData")
	mustWriteFile(t, filepath.Join(derived, "App-abc", "build.bin"), 3<<20)
	mustWriteFile(t, filepath.Join(home, "Downloads", "installer.dmg"), 2<<20)

	res, err := runScan(context.Background(), scanConfig{
		Roots:         []string{volume},
		Home:          home,
		Volume:        volume,
		Budget:        10 * time.Second,
		MinEntryBytes: 1 << 20,
	})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	byID := map[string]int64{}
	for _, category := range res.Categories {
		byID[category.ID] = category.Bytes
	}
	if byID["xcode"] != 3<<20 {
		t.Errorf("xcode bytes = %d, want %d (categories: %v)", byID["xcode"], 3<<20, byID)
	}
	if byID["downloads"] != 2<<20 {
		t.Errorf("downloads bytes = %d, want %d", byID["downloads"], 2<<20)
	}

	paths := map[string]bool{}
	for _, entry := range res.Entries {
		paths[entry.Path] = true
	}
	for _, want := range []string{"~/Library/Developer/Xcode/DerivedData", "~/Downloads"} {
		if !paths[want] {
			t.Errorf("entries missing %q (got %v)", want, paths)
		}
	}
}

func TestClassifierLongestPrefixWins(t *testing.T) {
	home := "/Users/me"
	classifier := newClassifier(home, "")

	cases := []struct {
		path        string
		category    string
		reclaimable bool
	}{
		{"/Users/me/Library/Developer/Xcode/DerivedData/App-abc", "xcode", true},
		{"/Users/me/Library/Developer/Xcode/Archives", "xcode", false},
		{"/Users/me/Library/Developer", "xcode", false},
		{"/Users/me/Library/Caches/Homebrew/downloads", "dev-cache", true},
		{"/Users/me/Library/Caches/com.spotify.client", "caches", true},
		{"/Users/me/Library/Caches", "caches", true},
		{"/Users/me/Downloads/installer.dmg", "downloads", false},
		{"/Users/me/Movies/trip.mov", "media", false},
		{"/Users/me/Projects/app/main.go", "other", false},
	}
	for _, testCase := range cases {
		category, reclaimable := classifier.classify(testCase.path)
		if category != testCase.category || reclaimable != testCase.reclaimable {
			t.Errorf("classify(%q) = (%s, %v), want (%s, %v)",
				testCase.path, category, reclaimable, testCase.category, testCase.reclaimable)
		}
	}
}

func TestScanCountsCategoriesAndDedupesHardlinks(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "Users", "me")
	mustWriteFile(t, filepath.Join(home, "Library", "Caches", "Big", "a.bin"), 3<<20)
	mustWriteFile(t, filepath.Join(home, "Library", "Caches", "Homebrew", "b.bin"), 2<<20)
	mustWriteFile(t, filepath.Join(home, "Downloads", "c.bin"), 1<<20)
	mustWriteFile(t, filepath.Join(home, "Movies", "d.bin"), 2<<20)
	mustWriteFile(t, filepath.Join(home, "Projects", "e.bin"), 4<<20)
	if err := os.Link(
		filepath.Join(home, "Library", "Caches", "Big", "a.bin"),
		filepath.Join(home, "Projects", "hard.bin"),
	); err != nil {
		t.Fatalf("link: %v", err)
	}

	result, err := runScan(context.Background(), scanConfig{
		Roots:         []string{root},
		Home:          home,
		MinEntryBytes: 1,
	})
	if err != nil {
		t.Fatalf("runScan: %v", err)
	}
	if result.Truncated {
		t.Fatal("small tree should not be truncated")
	}

	const wantScanned = 12 << 20
	if result.WalkedBytes != wantScanned {
		t.Errorf("WalkedBytes = %d, want %d (hardlink must count once)", result.WalkedBytes, wantScanned)
	}

	wantBytes := map[string]int64{
		"dev-cache": 2 << 20,
		"caches":    3 << 20,
		"media":     2 << 20,
		"downloads": 1 << 20,
		"other":     4 << 20,
	}
	var sum int64
	for _, category := range result.Categories {
		sum += category.Bytes
		if want, ok := wantBytes[category.ID]; ok && category.Bytes != want {
			t.Errorf("category %s bytes = %d, want %d", category.ID, category.Bytes, want)
		}
	}
	if sum != result.WalkedBytes {
		t.Errorf("category bytes sum = %d, want %d", sum, result.WalkedBytes)
	}

	paths := map[string]int64{}
	for _, entry := range result.Entries {
		paths[entry.Path] = entry.Bytes
	}
	for path, want := range map[string]int64{
		"~/Downloads":      1 << 20,
		"~/Movies":         2 << 20,
		"~/Library/Caches": 5 << 20,
	} {
		if paths[path] != want {
			t.Errorf("entry %q = %d, want %d (got %v)", path, paths[path], want, paths)
		}
	}
	for i := 1; i < len(result.Entries); i++ {
		if result.Entries[i-1].Bytes < result.Entries[i].Bytes {
			t.Fatalf("entries are not sorted by size: %+v", result.Entries)
		}
	}
}

func TestScanBudgetTruncates(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "a", "one.bin"), 1<<20)

	result, err := runScan(context.Background(), scanConfig{
		Roots:         []string{root},
		Home:          root,
		Budget:        time.Nanosecond,
		MinEntryBytes: 1,
	})
	if err != nil {
		t.Fatalf("a truncated scan must not fail: %v", err)
	}
	if !result.Truncated {
		t.Fatal("a nanosecond budget should truncate the scan")
	}
}

func TestScanHonoursCancellation(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "a", "one.bin"), 1<<20)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := runScan(ctx, scanConfig{Roots: []string{root}, Home: root}); err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestProgressPercentCapsAt99(t *testing.T) {
	cases := []struct {
		walked, used int64
		want         int
	}{
		{0, 100, 0},
		{45, 100, 50},
		{1000, 100, 99},
		{50, 0, 0},
	}
	for _, testCase := range cases {
		if got := progressPercent(testCase.walked, testCase.used); got != testCase.want {
			t.Errorf("progressPercent(%d, %d) = %d, want %d", testCase.walked, testCase.used, got, testCase.want)
		}
	}
}

func TestStoreAppendsPrunesAndDropsEntries(t *testing.T) {
	store := newAnalysisStore(t.TempDir())
	base := time.Date(2026, 11, 8, 0, 0, 0, 0, time.Local)
	for i := 0; i < analysisKeepEntries+3; i++ {
		report := Report{
			ID:         store.nextReportID(base.Add(time.Duration(i) * time.Minute)),
			Schema:     AnalysisSchema,
			ScannedAt:  base.Add(time.Duration(i) * time.Minute),
			Complete:   true,
			Categories: []Category{{ID: "caches", Label: "App caches", Bytes: int64(i) << 20}},
			Entries:    []Entry{{Category: "caches", Path: "~/Library/Caches", Bytes: 1 << 20}},
		}
		if err := store.append(report, "job"); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	meta, summaries, count := store.snapshot(0)
	if count != analysisKeepEntries+3 {
		t.Fatalf("count = %d, want %d", count, analysisKeepEntries+3)
	}
	if meta.LastCompletedAt == nil {
		t.Fatal("last_completed_at should be set")
	}
	if meta.LastAttempt == nil || meta.LastAttempt.Outcome != "completed" {
		t.Fatalf("last attempt = %+v, want completed", meta.LastAttempt)
	}

	for i, summary := range summaries {
		wantEntries := i < analysisKeepEntries
		if summary.HasEntries != wantEntries {
			t.Errorf("report %d has entries = %v, want %v", i, summary.HasEntries, wantEntries)
		}
		if !wantEntries && !summary.EntriesDropped {
			t.Errorf("report %d should be flagged entries_dropped", i)
		}
		if len(summary.Categories) != 1 {
			t.Errorf("report %d lost its category rollup", i)
		}
	}

	// Newest first.
	if summaries[0].ScannedAt.Before(summaries[1].ScannedAt) {
		t.Error("reports are not newest-first")
	}

	// The full report still has its rows only for the newest one.
	report, ok := store.report(summaries[0].ID)
	if !ok || len(report.Entries) != 1 {
		t.Fatalf("newest report entries = %d, want 1", len(report.Entries))
	}
	old, ok := store.report(summaries[len(summaries)-1].ID)
	if !ok || len(old.Entries) != 0 {
		t.Fatalf("oldest report should have dropped entries")
	}
}

func TestStoreRecoversInterruptedJob(t *testing.T) {
	dir := t.TempDir()
	store := newAnalysisStore(dir)
	store.writeJob(JobProgress{
		ID:        "job-1",
		StartedAt: time.Date(2026, 11, 8, 0, 1, 0, 0, time.Local),
		Phase:     "~/Library",
		State:     "running",
	})

	reloaded := newAnalysisStore(dir)
	meta, summaries, count := reloaded.snapshot(0)
	if count != 0 || len(summaries) != 0 {
		t.Fatalf("interrupted job must not create a report, got %d", count)
	}
	if meta.LastAttempt == nil || meta.LastAttempt.Outcome != "interrupted" {
		t.Fatalf("last attempt = %+v, want interrupted", meta.LastAttempt)
	}
	if _, err := os.Stat(filepath.Join(dir, "job.json")); !os.IsNotExist(err) {
		t.Error("stale job file should be removed after recovery")
	}
}

func TestManagerRunsScanAndSavesReport(t *testing.T) {
	manager := newTestManager(t)

	job, err := manager.Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if job.State != "running" {
		t.Fatalf("job state = %q, want running", job.State)
	}
	status := waitIdle(t, manager)

	if status.ReportCount != 1 {
		t.Fatalf("report count = %d, want 1", status.ReportCount)
	}
	if status.Latest == nil || status.Latest.UsedBytes != 6<<30 {
		t.Fatalf("latest = %+v, want used 6GiB", status.Latest)
	}
	if status.LastAttempt == nil || status.LastAttempt.Outcome != "completed" {
		t.Fatalf("last attempt = %+v, want completed", status.LastAttempt)
	}
	if status.LastCompletedAt == nil {
		t.Fatal("last_completed_at should be set")
	}

	report, ok := manager.Report(status.Latest.ID)
	if !ok {
		t.Fatal("report should be retrievable")
	}
	if !report.Complete || report.ScannedBytes != 5<<30 || len(report.Entries) != 1 {
		t.Fatalf("report = %+v", report)
	}
}

func TestManagerCancelWritesNoReport(t *testing.T) {
	manager := newAnalysisManager(newAnalysisStore(t.TempDir()))
	manager.now = time.Now
	manager.usage = func() (int64, int64) { return 6 << 30, 10 << 30 }
	manager.scan = func(ctx context.Context, cfg scanConfig) (scanResult, error) {
		cfg.Progress(1<<30, 10, 0, "~/Library")
		<-ctx.Done()
		return scanResult{}, ctx.Err()
	}

	if _, err := manager.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !manager.Cancel() {
		t.Fatal("Cancel should report the running scan")
	}
	status := waitIdle(t, manager)

	if status.ReportCount != 0 {
		t.Fatalf("cancelled scan must not write a report, got %d", status.ReportCount)
	}
	if status.LastAttempt == nil || status.LastAttempt.Outcome != "cancelled" {
		t.Fatalf("last attempt = %+v, want cancelled", status.LastAttempt)
	}
	if status.LastCompletedAt != nil {
		t.Fatal("cancelled scan must not set last_completed_at")
	}
	if manager.Cancel() {
		t.Fatal("Cancel with nothing running should be false")
	}
}

func TestManagerRejectsSecondStart(t *testing.T) {
	manager := newTestManager(t)
	manager.scan = func(ctx context.Context, cfg scanConfig) (scanResult, error) {
		<-ctx.Done()
		return scanResult{}, ctx.Err()
	}

	if _, err := manager.Start(); err != nil {
		t.Fatalf("first Start: %v", err)
	}
	_, err := manager.Start()
	if err != errAnalysisRunning {
		t.Fatalf("second Start error = %v, want errAnalysisRunning", err)
	}
	manager.Cancel()
	waitIdle(t, manager)
}

func TestAnalysisAPIStatusStartAndReports(t *testing.T) {
	manager := newTestManager(t)
	mux := http.NewServeMux()
	(&analysisAPI{manager: manager}).register(mux)

	// Status before the first run.
	body := doJSON(t, mux, http.MethodGet, "/api/server/disk/analysis/status", http.StatusOK)
	status, _ := body["running"].(bool)
	if status {
		t.Fatal("fresh manager should not be running")
	}
	if body["report_count"].(float64) != 0 {
		t.Fatalf("fresh report count = %v", body["report_count"])
	}

	// While a scan runs, a second start answers 409 with the running job so the
	// page can attach instead of queueing duplicate work.
	manager.scan = func(ctx context.Context, cfg scanConfig) (scanResult, error) {
		<-ctx.Done()
		return scanResult{}, ctx.Err()
	}
	doJSON(t, mux, http.MethodPost, "/api/server/disk/analysis/start", http.StatusOK)
	conflict := doJSON(t, mux, http.MethodPost, "/api/server/disk/analysis/start", http.StatusConflict)
	if conflict["job"] == nil {
		t.Fatalf("conflict response should carry the running job: %v", conflict)
	}
	if cancelled := doJSON(t, mux, http.MethodPost, "/api/server/disk/analysis/cancel", http.StatusOK); cancelled["cancelled"] != true {
		t.Fatalf("cancel while running = %v", cancelled["cancelled"])
	}
	if status := waitIdle(t, manager); status.ReportCount != 0 {
		t.Fatalf("cancelled scan wrote %d reports", status.ReportCount)
	}

	// A finished scan is listed and readable by id.
	manager.scan = func(ctx context.Context, cfg scanConfig) (scanResult, error) {
		return scanResult{
			WalkedBytes: 5 << 30,
			Categories:  []Category{{ID: "caches", Label: "App caches", Kind: "cache", Bytes: 5 << 30, Reclaimable: 5 << 30}},
			Entries:     []Entry{{Category: "caches", Path: "~/Library/Caches", Bytes: 5 << 30, Reclaimable: true}},
		}, nil
	}
	started := doJSON(t, mux, http.MethodPost, "/api/server/disk/analysis/start", http.StatusOK)
	if started["job"] == nil {
		t.Fatal("start should return its job")
	}
	waitIdle(t, manager)

	reports := doJSON(t, mux, http.MethodGet, "/api/server/disk/analysis/reports?limit=10", http.StatusOK)
	list, _ := reports["reports"].([]any)
	if len(list) != 1 {
		t.Fatalf("report list = %v", list)
	}
	id := list[0].(map[string]any)["id"].(string)
	detail := doJSON(t, mux, http.MethodGet, "/api/server/disk/analysis/reports/"+id, http.StatusOK)
	if detail["id"] != id {
		t.Fatalf("detail id = %v, want %s", detail["id"], id)
	}
	doJSON(t, mux, http.MethodGet, "/api/server/disk/analysis/reports/nope", http.StatusNotFound)

	cancel := doJSON(t, mux, http.MethodPost, "/api/server/disk/analysis/cancel", http.StatusOK)
	if cancel["cancelled"] != false {
		t.Fatalf("cancel with nothing running = %v", cancel["cancelled"])
	}
}

func newTestManager(t *testing.T) *analysisManager {
	t.Helper()
	manager := newAnalysisManager(newAnalysisStore(t.TempDir()))
	manager.now = time.Now
	manager.usage = func() (int64, int64) { return 6 << 30, 10 << 30 }
	manager.roots = func() []string { return nil }
	manager.scan = func(ctx context.Context, cfg scanConfig) (scanResult, error) {
		cfg.Progress(2<<30, 1000, 1, "~/Library/Caches")
		return scanResult{
			Roots:       []string{"/data"},
			WalkedBytes: 5 << 30,
			Files:       120000,
			Categories: []Category{
				{ID: "caches", Label: "App caches", Kind: "cache", Bytes: 3 << 30, Reclaimable: 3 << 30},
				{ID: "other", Label: "Everything else", Kind: "other", Bytes: 2 << 30},
			},
			Entries: []Entry{{Category: "caches", Path: "~/Library/Caches", Bytes: 3 << 30, Reclaimable: true}},
		}, nil
	}
	return manager
}

// waitIdle polls until the scan finishes, then returns the settled status.
func waitIdle(t *testing.T, manager *analysisManager) AnalysisStatus {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if status := manager.Status(); !status.Running {
			return status
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("scan did not finish")
	return AnalysisStatus{}
}

func doJSON(t *testing.T, handler http.Handler, method, target string, wantStatus int) map[string]any {
	t.Helper()
	request := httptest.NewRequest(method, target, nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != wantStatus {
		t.Fatalf("%s %s = %d, want %d (body %s)", method, target, recorder.Code, wantStatus, recorder.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("%s %s body: %v", method, target, err)
	}
	return body
}

func mustWriteFile(t *testing.T, path string, size int64) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	defer file.Close()
	if err := file.Truncate(size); err != nil {
		t.Fatalf("truncate %s: %v", path, err)
	}
}
