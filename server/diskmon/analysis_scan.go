package diskmon

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// scanConfig bounds one walk. The budget is a soft deadline: when it expires the
// walk stops and the report is marked truncated instead of being thrown away.
type scanConfig struct {
	Roots  []string
	Home   string
	Budget time.Duration
	// MinEntryBytes filters the directory list; 0 means the default.
	MinEntryBytes int64
	// Progress is called with cumulative counters, throttled to a few times a
	// second. It never runs while the walk holds its own lock.
	Progress func(walked, files int64, noAccess int, phase string)
	// Volume is a mount prefix stripped before paths are classified and shown,
	// so /System/Volumes/Data/Users/me/Library reads as ~/Library. Empty means
	// it is derived from Roots.
	Volume string
	// Workers caps how many directories are read at the same time; 0 picks a
	// default from GOMAXPROCS.
	Workers int
}

type scanResult struct {
	Categories   []Category
	Entries      []Entry
	Roots        []string
	WalkedBytes  int64
	Files        int64
	NoAccessDirs int
	Truncated    bool
	// Skipped lists the mounts and folders the walk refused to enter, so the
	// report can say which parts of the disk it never looked at.
	Skipped []SkippedPath
	// PriorityRoots are the subtrees the category numbers lead with. Priority
	// roots finish before the rest of the volume, so even a truncated report
	// has a trustworthy headline.
	PriorityRoots   []string
	PriorityScanned int
}

// skipDirNames are never descended into: they hold no user data, or would count
// the same bytes twice on macOS.
var skipDirNames = map[string]bool{
	".Spotlight-V100":         true,
	".fseventsd":              true,
	".DocumentRevisions-V100": true,
	".TemporaryItems":         true,
	".Trashes":                true,
	"Trashes":                 true,
}

// skipPaths are absolute paths in the classifier's own space ("/" and the home
// directory), never descended into. A non-empty reason means the skip costs
// coverage and is worth reporting; the autofs mounts and the cloud File
// Provider folder are the ones that matter, because a single readdir there can
// block for minutes and no time budget survives that.
var skipPaths = map[string]string{
	"/Volumes":                "", // other mounted volumes are counted on their own
	"/private/var/vm":         "", // swap files
	"/Backups.backupdb":       "",
	"/.MobileBackups":         "",
	"/System/Volumes/Data":    "", // firmlink loop guard
	"/System/Volumes/Update":  "", // staged OS updates
	"/System/Volumes/Preboot": "",
	"/System/Volumes/VM":      "",
	"/home":                   "autofs mount",
	"/net":                    "autofs mount",
}

// isCloudDrive matches a cloud File Provider root. Its content lives behind a
// network round-trip even when nothing is cached, so listing it can block for
// minutes.
func isCloudDrive(logical string) bool {
	return strings.HasPrefix(logical, "/Users/") && strings.HasSuffix(logical, "/Library/CloudStorage")
}

// priorityRootPaths are the subtrees the page leads with: the user's own
// folders plus the category roots that live outside them. They are scheduled
// before the rest of the volume.
func priorityRootPaths(home string) []string {
	roots := []string{
		"/Library/Developer",
		"/Library/Caches",
		"/Applications",
		"/private/var/folders",
		"/opt",
	}
	if home != "" && home != "/" {
		roots = append([]string{home}, roots...)
	}
	return roots
}

type scanNode struct {
	path        string
	parent      *scanNode
	cat         string
	reclaimable bool
	record      bool
	// priorityRoot marks a node that was seeded for the priority set; its size
	// feeds the coverage counters instead of a parent.
	priorityRoot bool

	// size, reclaim, pending and recorded are guarded by scanAccumulator.mu.
	size     int64
	reclaim  int64
	pending  int
	recorded bool
	// abandoned marks a directory whose scan was given up on, so a read stuck
	// in the kernel can never publish its bytes after the fact.
	abandoned bool
}

type scanAccumulator struct {
	cfg      scanConfig
	class    *classifier
	ctx      context.Context
	deadline time.Time
	queue    *scanQueue

	mu         sync.Mutex
	catBytes   map[string]int64
	catReclaim map[string]int64
	recorded   []*scanNode
	links      map[[2]uint64]*parkedLink
	visited    map[string]bool
	skipped    []SkippedPath
	skippedAt  map[string]bool
	phase      string

	walked   int64
	files    int64
	noAccess int

	// priorityList is the priority set and priorityNodes are its tree roots: a
	// root whose pending count reached zero was read completely.
	priorityList  []string
	priorityNodes []*scanNode

	stop      atomic.Bool
	truncated atomic.Bool
}

// seed queues the priority subtrees, then the scan roots, and finally releases
// the queue's startup hold so blocked workers can start popping. Priority roots
// go in first, and every child of a priority task keeps the flag, so the folders
// the page leads with are finished before the rest of the volume is listed.
func (a *scanAccumulator) seed() {
	// The deques are LIFO, so priority roots are queued in reverse: the home
	// directory, the one the page leads with, has to come back off the stack
	// first.
	priority := append([]string{}, a.priorityList...)
	nodes := make([]*scanNode, len(priority))
	for i := len(priority) - 1; i >= 0; i-- {
		nodes[i] = a.seedDir(physicalPath(priority[i], a.cfg.Volume), true)
	}
	scanned := make([]string, 0, len(priority))
	roots := make([]*scanNode, 0, len(priority))
	for i, logical := range priority {
		if nodes[i] == nil {
			continue
		}
		scanned = append(scanned, shortenHome(logical, a.cfg.Home))
		roots = append(roots, nodes[i])
	}
	a.mu.Lock()
	a.priorityList = scanned
	a.priorityNodes = roots
	a.mu.Unlock()

	for _, root := range a.cfg.Roots {
		a.seedDir(filepath.Clean(root), false)
	}
	a.queue.release()
}

// seedDir queues one directory as a fresh tree root. It reports whether the
// directory was queued; missing paths and paths another tree already owns are
// skipped, so nothing is counted twice.
func (a *scanAccumulator) seedDir(path string, priorityRoot bool) *scanNode {
	if !a.withinRoots(path) {
		return nil
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() {
		return nil
	}
	node := &scanNode{path: path, cat: "other", pending: 1, priorityRoot: priorityRoot}
	node.cat, node.reclaimable = a.class.classify(path)
	node.record = a.shouldRecord(path)

	a.mu.Lock()
	if a.visited[path] {
		a.mu.Unlock()
		return nil
	}
	a.visited[path] = true
	a.mu.Unlock()

	a.queue.push(&dirTask{node: node, priority: priorityRoot})
	return node
}

// work pulls directories until the queue is empty or the walk was stopped.
func (a *scanAccumulator) work() {
	for {
		task, ok := a.queue.pop()
		if !ok {
			return
		}
		a.scanDirWithin(task)
		a.queue.done()
	}
}

// expired reports whether the walk must stop. The first goroutine to notice
// stops the queue, so the others drain their current directory instead of all
// racing the same deadline.
func (a *scanAccumulator) expired() bool {
	if a.stop.Load() {
		return true
	}
	err := a.ctx.Err()
	if err == nil && !time.Now().After(a.deadline) {
		return false
	}
	if a.stop.CompareAndSwap(false, true) {
		if err == nil {
			a.truncated.Store(true)
		}
		a.queue.stop()
	}
	return true
}

func (a *scanAccumulator) setPhase(path string) {
	phase := shortenHome(a.class.logical(path), a.cfg.Home)
	a.mu.Lock()
	a.phase = phase
	a.mu.Unlock()
}

// reportLoop pushes counters to the caller a few times a second. The walk only
// bumps numbers; no directory read ever waits on this goroutine.
func (a *scanAccumulator) reportLoop(stop <-chan struct{}) {
	if a.cfg.Progress == nil {
		return
	}
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			a.mu.Lock()
			walked, files, noAccess, phase := a.walked, a.files, a.noAccess, a.phase
			a.mu.Unlock()
			a.cfg.Progress(walked, files, noAccess, phase)
		}
	}
}

// priorityScannedLocked counts the priority roots whose subtree finished. The
// caller holds a.mu, or is the single goroutine still running.
func (a *scanAccumulator) priorityScannedLocked() int {
	scanned := 0
	for _, node := range a.priorityNodes {
		if node.pending == 0 {
			scanned++
		}
	}
	return scanned
}

func (a *scanAccumulator) finish() scanResult {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.attributeLinks()
	// Skipped paths surface in the report and must not depend on which worker
	// finished first, so they are ordered here instead of at discovery time.
	sort.Slice(a.skipped, func(i, j int) bool { return a.skipped[i].Path < a.skipped[j].Path })

	res := scanResult{
		Roots:           append([]string{}, a.cfg.Roots...),
		WalkedBytes:     a.walked,
		Files:           a.files,
		NoAccessDirs:    a.noAccess,
		Truncated:       a.truncated.Load(),
		Skipped:         append([]SkippedPath{}, a.skipped...),
		PriorityRoots:   append([]string{}, a.priorityList...),
		PriorityScanned: a.priorityScannedLocked(),
	}
	res.Categories = make([]Category, 0, len(analysisCategories))
	for _, def := range analysisCategories {
		res.Categories = append(res.Categories, Category{
			ID:          def.ID,
			Label:       def.Label,
			Kind:        def.Kind,
			Bytes:       a.catBytes[def.ID],
			Reclaimable: a.catReclaim[def.ID],
		})
	}
	res.Entries = a.topEntries()
	for i := range res.Categories {
		cat := &res.Categories[i]
		for _, entry := range res.Entries {
			if entry.Category != cat.ID || len(cat.Paths) >= 4 {
				continue
			}
			if !contains(cat.Paths, entry.Path) {
				cat.Paths = append(cat.Paths, entry.Path)
			}
		}
	}
	return res
}

// shouldRecord picks the directories worth showing: category roots, one level
// into them, and the user's own top-level folders. Recording every ancestor
// would bury the list under "Users".
func (a *scanAccumulator) shouldRecord(path string) bool {
	if a.class.isRuleRoot(path) {
		return true
	}
	parent := filepath.Dir(a.class.logical(path))
	if parent == filepath.Clean(a.cfg.Home) {
		return true
	}
	return a.class.isRuleRoot(parent)
}

// topEntries keeps the biggest recorded directories. A parent that is almost
// entirely made of one child is skipped, so "~/Library/Caches" does not hide
// "~/Library/Caches/Homebrew".
//
// The caller holds the accumulator lock and every worker has stopped, so this
// runs on a frozen snapshot. Sizes tie often (empty folders), hence the path
// tiebreak: the parallel walk closes directories in whatever order the workers
// finish them, and the report must not depend on that order.
func (a *scanAccumulator) topEntries() []Entry {
	minBytes := a.cfg.MinEntryBytes
	if minBytes <= 0 {
		minBytes = 256 << 20
	}
	sorted := append([]*scanNode{}, a.recorded...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].size != sorted[j].size {
			return sorted[i].size > sorted[j].size
		}
		return sorted[i].path < sorted[j].path
	})

	kept := make([]*scanNode, 0, 40)
	for _, node := range sorted {
		if len(kept) >= 40 {
			break
		}
		if node.size < minBytes {
			continue
		}
		// A parent that is only a wrapper around this child is dropped in
		// favour of the child, so "~/Library" cannot hide
		// "~/Library/Developer/Xcode/DerivedData". Category roots always keep
		// their row, and so do parents built from several children.
		for i := 0; i < len(kept); i++ {
			parent := kept[i]
			if a.class.isRuleRoot(parent.path) {
				continue
			}
			if strings.HasPrefix(node.path, parent.path+"/") && parent.size-node.size < parent.size/10 {
				kept = append(kept[:i], kept[i+1:]...)
				i--
			}
		}
		kept = append(kept, node)
	}

	out := make([]Entry, 0, len(kept))
	for _, node := range kept {
		out = append(out, Entry{
			Category:    node.cat,
			Path:        shortenHome(a.class.logical(node.path), a.cfg.Home),
			Bytes:       node.size,
			Reclaimable: node.reclaimable,
		})
	}
	return out
}

func contains(list []string, want string) bool {
	for _, item := range list {
		if item == want {
			return true
		}
	}
	return false
}

// volumePrefix reports the mount prefix that should be dropped from walked
// paths, or "" when every root is already rooted at "/".
func volumePrefix(roots []string) string {
	for _, root := range roots {
		root = filepath.Clean(root)
		if root == dataMount || strings.HasPrefix(root, dataMount+"/") {
			return dataMount
		}
	}
	return ""
}

// defaultScanRoots is the writable half of the boot volume: the data volume on
// macOS, "/" elsewhere. Reading it needs no sudo.
func defaultScanRoots() []string {
	if info, err := os.Stat(dataMount); err == nil && info.IsDir() {
		return []string{dataMount}
	}
	return []string{"/"}
}
