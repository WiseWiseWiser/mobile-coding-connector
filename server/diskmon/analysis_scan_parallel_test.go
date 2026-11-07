package diskmon

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

// buildParallelFixture lays out a tree that exercises every way a parallel walk
// can disagree with a serial one: sibling directories at the same depth, hard
// links across different subtrees, an empty directory, a symlink loop and a
// directory that cannot be read.
func buildParallelFixture(t *testing.T) (root, home, locked string) {
	t.Helper()
	root = t.TempDir()
	home = filepath.Join(root, "Users", "me")

	mustWriteFile(t, filepath.Join(home, "Library", "Caches", "Homebrew", "brew.bin"), 5<<20)
	mustWriteFile(t, filepath.Join(home, "Library", "Caches", "Big", "cache.bin"), 3<<20)
	mustWriteFile(t, filepath.Join(home, "Downloads", "installer.dmg"), 2<<20)
	mustWriteFile(t, filepath.Join(home, "Movies", "clip.mov"), 4<<20)
	mustWriteFile(t, filepath.Join(home, "Projects", "app", "main.bin"), 1<<20)
	mustWriteFile(t, filepath.Join(root, "Applications", "Tool.app", "blob.bin"), 6<<20)

	// Hard links living in subtrees that different workers pull at the same
	// time; whichever link wins, the inode must be counted once.
	source := filepath.Join(home, "Library", "Caches", "Big", "cache.bin")
	for _, target := range []string{
		filepath.Join(home, "Projects", "app", "hard.bin"),
		filepath.Join(root, "Applications", "Tool.app", "hard.bin"),
	} {
		if err := os.Link(source, target); err != nil {
			t.Fatalf("link: %v", err)
		}
	}

	empty := filepath.Join(home, "Empty")
	if err := os.MkdirAll(empty, 0o755); err != nil {
		t.Fatalf("mkdir empty: %v", err)
	}
	if err := os.Symlink(home, filepath.Join(empty, "loop")); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	locked = filepath.Join(root, "private", "locked")
	if err := os.MkdirAll(locked, 0o755); err != nil {
		t.Fatalf("mkdir locked: %v", err)
	}
	mustWriteFile(t, filepath.Join(locked, "hidden.bin"), 7<<20)
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0o755) })
	return root, home, locked
}

// TestScanParallelMatchesSerial is the golden test: eight workers must produce
// the same report as one. Any order dependence in the folds, in the hardlink
// accounting or in the entry pruning shows up here as a diff.
func TestScanParallelMatchesSerial(t *testing.T) {
	root, home, _ := buildParallelFixture(t)

	scan := func(workers int) scanResult {
		result, err := runScan(context.Background(), scanConfig{
			Roots:         []string{root},
			Home:          home,
			Volume:        root,
			Budget:        30 * time.Second,
			MinEntryBytes: 1,
			Workers:       workers,
		})
		if err != nil {
			t.Fatalf("%d workers: %v", workers, err)
		}
		return result
	}

	serial := scan(1)
	parallel := scan(8)

	if partial := scan(3); !reflect.DeepEqual(partial.Categories, parallel.Categories) {
		t.Error("three workers disagree with eight on the category rollup")
	}

	if !reflect.DeepEqual(serial.Categories, parallel.Categories) {
		t.Errorf("categories differ\n serial: %+v\n parallel: %+v", serial.Categories, parallel.Categories)
	}
	if !reflect.DeepEqual(serial.Entries, parallel.Entries) {
		t.Errorf("entries differ\n serial: %+v\n parallel: %+v", serial.Entries, parallel.Entries)
	}
	if serial.WalkedBytes != parallel.WalkedBytes || serial.Files != parallel.Files {
		t.Errorf("totals differ: serial %d/%d, parallel %d/%d",
			serial.WalkedBytes, serial.Files, parallel.WalkedBytes, parallel.Files)
	}
	if serial.NoAccessDirs != parallel.NoAccessDirs {
		t.Errorf("unreadable dirs: serial %d, parallel %d", serial.NoAccessDirs, parallel.NoAccessDirs)
	}
	if !reflect.DeepEqual(serial.Skipped, parallel.Skipped) {
		t.Errorf("skipped paths differ: serial %+v, parallel %+v", serial.Skipped, parallel.Skipped)
	}

	// The hard links must have been counted exactly once.
	const files = 6
	want := int64((5 + 3 + 2 + 4 + 1 + 6) << 20)
	if parallel.WalkedBytes != want {
		t.Errorf("walked = %d, want %d (%d files, hard links counted once)", parallel.WalkedBytes, want, files)
	}
	var sum int64
	for _, category := range parallel.Categories {
		sum += category.Bytes
	}
	if sum != parallel.WalkedBytes {
		t.Errorf("category sum = %d, walked = %d", sum, parallel.WalkedBytes)
	}
	if parallel.NoAccessDirs != 1 {
		t.Errorf("unreadable dirs = %d, want 1", parallel.NoAccessDirs)
	}
}

// TestScanParallelIsRepeatable guards against a report that changes between
// runs on the same unchanged tree: the trend page compares reports by value.
func TestScanParallelIsRepeatable(t *testing.T) {
	root, home, _ := buildParallelFixture(t)

	var first string
	for attempt := 0; attempt < 4; attempt++ {
		result, err := runScan(context.Background(), scanConfig{
			Roots:         []string{root},
			Home:          home,
			Budget:        30 * time.Second,
			MinEntryBytes: 1,
			Workers:       8,
		})
		if err != nil {
			t.Fatalf("run %d: %v", attempt, err)
		}
		encoded, err := json.Marshal(result.Categories)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		next := string(encoded) + "|" + marshalOrFail(t, result)
		if attempt == 0 {
			first = next
			continue
		}
		if next != first {
			t.Fatalf("run %d differs from run 0", attempt)
		}
	}
}

func marshalOrFail(t *testing.T, result scanResult) string {
	t.Helper()
	entries, err := json.Marshal(result.Entries)
	if err != nil {
		t.Fatalf("marshal entries: %v", err)
	}
	return string(entries)
}

// TestScanPriorityCoverage checks the contract behind the "core folders
// scanned" chip: the home directory is in the priority set, it is scanned
// before anything else, and it is reported as complete once it finishes.
func TestScanPriorityCoverage(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "Users", "me")
	mustWriteFile(t, filepath.Join(home, "Downloads", "a.bin"), 1<<20)
	mustWriteFile(t, filepath.Join(root, "usr", "share", "b.bin"), 2<<20)

	result, err := runScan(context.Background(), scanConfig{
		Roots:         []string{root},
		Home:          home,
		Volume:        root,
		Budget:        10 * time.Second,
		MinEntryBytes: 1,
		Workers:       4,
	})
	if err != nil {
		t.Fatalf("runScan: %v", err)
	}

	if len(result.PriorityRoots) != 1 || result.PriorityRoots[0] != "~" {
		t.Fatalf("priority roots = %v, want [~]", result.PriorityRoots)
	}
	if result.PriorityScanned != 1 {
		t.Errorf("priority scanned = %d, want 1", result.PriorityScanned)
	}

	coverage := scanCoverage(result)
	if coverage == nil || !coverage.PriorityComplete {
		t.Fatalf("coverage = %+v, want the priority set complete", coverage)
	}
}

// TestScanSeedsPriorityRootsLast proves the ordering: the deques are LIFO, so
// the home directory is pushed last in order to come off the stack first.
func TestScanSeedsPriorityRootsLast(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "Users", "me")
	mustWriteFile(t, filepath.Join(home, "Downloads", "a.bin"), 1<<20)
	mustWriteFile(t, filepath.Join(root, "usr", "share", "b.bin"), 2<<20)

	// runScan normalizes the home directory into logical space, so the
	// accumulator sees "/Users/me" rather than the temporary volume prefix.
	acc := &scanAccumulator{
		cfg:          scanConfig{Roots: []string{root}, Home: "/Users/me", Volume: root},
		class:        newClassifier("/Users/me", root),
		deadline:     time.Now().Add(time.Minute),
		queue:        newScanQueue(),
		catBytes:     map[string]int64{},
		catReclaim:   map[string]int64{},
		links:        map[[2]uint64]*parkedLink{},
		visited:      map[string]bool{},
		skippedAt:    map[string]bool{},
		priorityList: priorityRootPaths("/Users/me"),
	}
	acc.seed()

	task, ok := acc.queue.pop()
	if !ok {
		t.Fatal("seeding queued nothing")
	}
	if want := filepath.Join(root, "Users", "me"); task.node.path != want {
		t.Fatalf("first task = %s, want %s", task.node.path, want)
	}
	if !task.priority {
		t.Error("the home directory was not flagged as priority work")
	}
	acc.queue.done()
	acc.queue.stop()
}

// TestScanSkipsAutofsAndCloudDrives keeps the walk out of the paths that block
// for minutes: the autofs mounts and the cloud File Provider roots.
func TestScanSkipsAutofsAndCloudDrives(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "Users", "me")
	mustWriteFile(t, filepath.Join(root, "home", "ghost", "a.bin"), 1<<20)
	mustWriteFile(t, filepath.Join(root, "net", "ghost", "b.bin"), 1<<20)
	mustWriteFile(t, filepath.Join(home, "Library", "CloudStorage", "Drive", "c.bin"), 1<<20)
	mustWriteFile(t, filepath.Join(home, "Downloads", "d.bin"), 1<<20)

	result, err := runScan(context.Background(), scanConfig{
		Roots:         []string{root},
		Home:          home,
		Volume:        root,
		Budget:        10 * time.Second,
		MinEntryBytes: 1,
		Workers:       4,
	})
	if err != nil {
		t.Fatalf("runScan: %v", err)
	}

	want := int64(1 << 20)
	if result.WalkedBytes != want {
		t.Errorf("walked = %d, want %d: a skipped mount was walked", result.WalkedBytes, want)
	}

	reasons := map[string]string{}
	for _, skip := range result.Skipped {
		reasons[skip.Path] = skip.Reason
	}
	for path, reason := range map[string]string{
		"/home":                  "autofs mount",
		"/net":                   "autofs mount",
		"~/Library/CloudStorage": "cloud drive",
	} {
		if got, ok := reasons[path]; !ok || got != reason {
			t.Errorf("skipped[%q] = %q (present=%v), want %q", path, got, ok, reason)
		}
	}
}

// TestScanQueueServesPriorityFirst pins the scheduler: a priority directory is
// handed out before any low-priority one, no matter the push order.
func TestScanQueueServesPriorityFirst(t *testing.T) {
	queue := newScanQueue()
	queue.release()
	low := &dirTask{node: &scanNode{path: "/low"}}
	high := &dirTask{node: &scanNode{path: "/high"}, priority: true}
	queue.push(low)
	queue.push(high)

	task, ok := queue.pop()
	if !ok || task != high {
		t.Fatalf("first task = %+v, want the priority one", task)
	}
	queue.done()

	task, ok = queue.pop()
	if !ok || task != low {
		t.Fatalf("second task = %+v, want the low-priority one", task)
	}
	queue.done()
}

// TestScanQueueStopsOnEmpty proves the workers exit instead of hanging once the
// last directory is done.
func TestScanQueueStopsOnEmpty(t *testing.T) {
	queue := newScanQueue()
	queue.release()

	finished := make(chan struct{})
	go func() {
		if _, ok := queue.pop(); ok {
			t.Error("an empty queue handed out a task")
		}
		close(finished)
	}()
	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		t.Fatal("pop blocked on an empty queue")
	}
}

// TestScanSkipDirs covers the ignore table itself, including the entries that
// are skipped without being reported.
func TestScanSkipDirs(t *testing.T) {
	acc := &scanAccumulator{cfg: scanConfig{Home: "/Users/me"}}

	cases := []struct {
		logical string
		name    string
		skip    bool
		reason  string
	}{
		{"/Users/me/Library/Caches/Homebrew", "Homebrew", false, ""},
		{"/Users/me/Library/CloudStorage", "CloudStorage", true, "cloud drive"},
		{"/home", "home", true, "autofs mount"},
		{"/net", "net", true, "autofs mount"},
		{"/Volumes", "Volumes", true, ""},
		{"/private/var/vm", "vm", true, ""},
		{"/.Spotlight-V100", ".Spotlight-V100", true, ""},
		{"/Users/me/.Trash", ".Trash", false, ""},
	}
	for _, testCase := range cases {
		reason, skip := acc.skipDir(testCase.logical, testCase.name)
		if skip != testCase.skip || reason != testCase.reason {
			t.Errorf("skipDir(%q) = (%q, %v), want (%q, %v)",
				testCase.logical, reason, skip, testCase.reason, testCase.skip)
		}
	}
}

// TestScanDefaultWorkersIsBounded documents the cap: the walk is syscall bound
// and more than a handful of readers stopped paying off.
func TestScanDefaultWorkersIsBounded(t *testing.T) {
	if got := defaultScanWorkers(); got < 1 || got > 8 {
		t.Errorf("defaultScanWorkers() = %d, want between 1 and 8", got)
	}
}

// TestScanTruncatedKeepsInvariant holds the report invariants under a budget
// that cuts the walk in half.
func TestScanTruncatedKeepsInvariant(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "Users", "me")
	for i := 0; i < 60; i++ {
		mustWriteFile(t, filepath.Join(home, "Projects", "p", "f.bin"), 1<<20)
	}

	result, err := runScan(context.Background(), scanConfig{
		Roots:         []string{root},
		Home:          home,
		Volume:        root,
		Budget:        time.Nanosecond,
		MinEntryBytes: 1,
		Workers:       4,
	})
	if err != nil {
		t.Fatalf("a truncated scan must not fail: %v", err)
	}
	if !result.Truncated {
		t.Fatal("a nanosecond budget should truncate")
	}
	var sum int64
	for _, category := range result.Categories {
		sum += category.Bytes
	}
	if sum != result.WalkedBytes {
		t.Errorf("category sum = %d, walked = %d", sum, result.WalkedBytes)
	}
	if result.PriorityScanned >= 1 && result.PriorityRoots == nil {
		t.Error("priority scanned without a priority root list")
	}
}

func TestWaitWithin(t *testing.T) {
	pending := make(chan struct{})
	if waitWithin(pending, 20*time.Millisecond) {
		t.Fatal("waitWithin reported a pending channel as done")
	}
	close(pending)
	if !waitWithin(pending, time.Second) {
		t.Fatal("waitWithin missed a closed channel")
	}
}

// TestScanAbandonDropsTheDirectory covers the stall guard: a directory that
// stops making progress is reported once as not counted, and the bytes a stuck
// read publishes later are ignored.
func TestScanAbandonDropsTheDirectory(t *testing.T) {
	acc := &scanAccumulator{
		cfg:        scanConfig{Home: "/Users/me"},
		class:      newClassifier("/Users/me", ""),
		queue:      newScanQueue(),
		catBytes:   map[string]int64{},
		catReclaim: map[string]int64{},
		links:      map[[2]uint64]*parkedLink{},
		visited:    map[string]bool{},
		skippedAt:  map[string]bool{},
	}
	parent := &scanNode{pending: 1}
	node := &scanNode{
		path:    "/Users/me/Library/Containers/stuck/Data",
		parent:  parent,
		pending: 1,
		cat:     "other",
		record:  true,
	}

	acc.abandon(node)
	acc.abandon(node)
	if len(acc.skipped) != 1 {
		t.Fatalf("skipped = %v, want one entry", acc.skipped)
	}
	if got := acc.skipped[0].Path; got != "~/Library/Containers/stuck/Data" {
		t.Fatalf("skipped path = %q", got)
	}
	if got := acc.skipped[0].Reason; got != "read timed out" {
		t.Fatalf("skipped reason = %q", got)
	}

	// The stuck goroutine comes back after the fact: nothing may be published.
	acc.settle(node, 4096, 2, nil, nil, false, false)
	if acc.walked != 0 || acc.files != 0 || len(acc.recorded) != 0 || parent.pending != 0 {
		t.Fatalf("late settle leaked: walked=%d files=%d recorded=%d pending=%d",
			acc.walked, acc.files, len(acc.recorded), parent.pending)
	}
}

// A wedged directory must not take its parent with it: ~/Library has a few
// unreadable subtrees, but dropping ~/Library would hide ~/Library/Caches.
func TestStallKeepsTheParent(t *testing.T) {
	acc := &scanAccumulator{
		cfg:       scanConfig{Home: "/Users/me"},
		class:     newClassifier("/Users/me", ""),
		queue:     newScanQueue(),
		visited:   map[string]bool{},
		skippedAt: map[string]bool{},
	}
	parent := &scanNode{path: "/Users/me/Library", pending: 1}
	node := &scanNode{
		path:    "/Users/me/Library/Containers/stuck",
		parent:  parent,
		pending: 1,
		cat:     "other",
	}

	acc.stall(node)

	if !node.abandoned {
		t.Fatal("stalled directory was not abandoned")
	}
	if parent.abandoned {
		t.Fatal("stall escalated to the parent directory")
	}
	if len(acc.skipped) != 1 {
		t.Fatalf("skipped = %+v, want only the stalled directory", acc.skipped)
	}
}
