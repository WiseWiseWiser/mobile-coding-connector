package diskmon

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
	// Progress is called from the walk goroutine with cumulative counters.
	Progress func(walked, files int64, noAccess int, phase string)
	// Volume is a mount prefix stripped before paths are classified and shown,
	// so /System/Volumes/Data/Users/me/Library reads as ~/Library. Empty means
	// it is derived from Roots.
	Volume string
}

var errScanTruncated = errors.New("scan budget exceeded")

type scanResult struct {
	Categories   []Category
	Entries      []Entry
	Roots        []string
	WalkedBytes  int64
	Files        int64
	NoAccessDirs int
	Truncated    bool
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

// skipRelPaths are skipped relative to a scan root.
var skipRelPaths = map[string]bool{
	"Volumes":                true, // other mounted volumes are counted on their own
	"private/var/vm":         true, // swap files
	"Backups.backupdb":       true,
	".MobileBackups":         true,
	"System/Volumes/Data":    true, // firmlink loop guard
	"System/Volumes/Update":  true, // staged OS updates
	"System/Volumes/Preboot": true,
	"System/Volumes/VM":      true,
}

type scanFrame struct {
	path        string
	cat         string
	reclaimable bool
	record      bool
	size        int64
}

type scanAccumulator struct {
	cfg      scanConfig
	class    *classifier
	deadline time.Time

	catBytes   map[string]int64
	catReclaim map[string]int64
	recorded   []*scanFrame
	recordedAt map[string]bool
	frames     []*scanFrame
	seenLinks  map[[2]uint64]bool

	walked     int64
	files      int64
	noAccess   int
	lastEmit   time.Time
	filesSince int
	truncated  bool
}

func runScan(ctx context.Context, cfg scanConfig) (scanResult, error) {
	if cfg.Budget <= 0 {
		cfg.Budget = 120 * time.Second
	}
	if cfg.Home == "" {
		cfg.Home, _ = os.UserHomeDir()
	}
	if cfg.Volume == "" {
		cfg.Volume = volumePrefix(cfg.Roots)
	}
	// Home is compared against walked paths, so it lives in the same space:
	// with the volume prefix dropped.
	cfg.Home = logicalPath(cfg.Home, cfg.Volume)
	acc := &scanAccumulator{
		cfg:        cfg,
		class:      newClassifier(cfg.Home, cfg.Volume),
		deadline:   time.Now().Add(cfg.Budget),
		catBytes:   map[string]int64{},
		catReclaim: map[string]int64{},
		recordedAt: map[string]bool{},
		seenLinks:  map[[2]uint64]bool{},
	}

	for _, root := range cfg.Roots {
		if err := acc.walkRoot(ctx, root); err != nil {
			if errors.Is(err, errScanTruncated) {
				acc.truncated = true
				break
			}
			return acc.finish(), err
		}
	}
	return acc.finish(), nil
}

func (a *scanAccumulator) walkRoot(ctx context.Context, root string) error {
	root = filepath.Clean(root)
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() {
		return nil
	}

	a.frames = []*scanFrame{{path: root, cat: "other"}}

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if walkErr != nil {
			if path != root && os.IsPermission(walkErr) {
				a.noAccess++
			}
			return nil
		}
		if path == root {
			return nil
		}

		// Depth-first order: drop frames this path has left behind.
		for len(a.frames) > 1 && !strings.HasPrefix(path, a.frames[len(a.frames)-1].path+"/") {
			a.popFrame()
		}

		if d.IsDir() {
			if a.skipDir(root, path, d.Name()) {
				return fs.SkipDir
			}
			if time.Now().After(a.deadline) {
				return errScanTruncated
			}
			cat, reclaimable := a.class.classify(path)
			frame := &scanFrame{
				path:        path,
				cat:         cat,
				reclaimable: reclaimable,
				record:      a.shouldRecord(path),
			}
			if frame.record && !a.recordedAt[path] {
				a.recordedAt[path] = true
				a.recorded = append(a.recorded, frame)
			}
			a.frames = append(a.frames, frame)
			a.emit(path, false)
			return nil
		}

		if !d.Type().IsRegular() {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return nil
		}
		a.files++
		if dev, ino, ok := hardlinkKey(fi); ok {
			key := [2]uint64{dev, ino}
			if a.seenLinks[key] {
				return nil
			}
			a.seenLinks[key] = true
		}
		size := fi.Size()
		if size > 0 {
			frame := a.frames[len(a.frames)-1]
			frame.size += size
			a.catBytes[frame.cat] += size
			if frame.reclaimable {
				a.catReclaim[frame.cat] += size
			}
			a.walked += size
		}

		a.filesSince++
		if a.filesSince >= 4000 {
			a.filesSince = 0
			if time.Now().After(a.deadline) {
				return errScanTruncated
			}
			a.emit(a.frames[len(a.frames)-1].path, true)
		}
		return nil
	})
	for len(a.frames) > 1 {
		a.popFrame()
	}
	a.frames = nil
	return err
}

// popFrame closes the innermost directory and folds its size into its parent, so
// recorded ancestors end up with the cumulative total they represent.
func (a *scanAccumulator) popFrame() {
	last := a.frames[len(a.frames)-1]
	a.frames = a.frames[:len(a.frames)-1]
	if len(a.frames) > 0 {
		a.frames[len(a.frames)-1].size += last.size
	}
}

func (a *scanAccumulator) skipDir(root, path, name string) bool {
	if skipDirNames[name] {
		return true
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return skipRelPaths[filepath.ToSlash(rel)]
}

func (a *scanAccumulator) emit(path string, force bool) {
	if a.cfg.Progress == nil {
		return
	}
	now := time.Now()
	if !force && now.Sub(a.lastEmit) < 400*time.Millisecond {
		return
	}
	a.lastEmit = now
	a.cfg.Progress(a.walked, a.files, a.noAccess, shortenHome(a.class.logical(path), a.cfg.Home))
}

func (a *scanAccumulator) finish() scanResult {
	res := scanResult{
		Roots:        append([]string{}, a.cfg.Roots...),
		WalkedBytes:  a.walked,
		Files:        a.files,
		NoAccessDirs: a.noAccess,
		Truncated:    a.truncated,
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
func (a *scanAccumulator) topEntries() []Entry {
	minBytes := a.cfg.MinEntryBytes
	if minBytes <= 0 {
		minBytes = 256 << 20
	}
	sorted := append([]*scanFrame{}, a.recorded...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].size > sorted[j].size })

	kept := make([]*scanFrame, 0, 40)
	for _, frame := range sorted {
		if len(kept) >= 40 {
			break
		}
		if frame.size < minBytes {
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
			if strings.HasPrefix(frame.path, parent.path+"/") && parent.size-frame.size < parent.size/10 {
				kept = append(kept[:i], kept[i+1:]...)
				i--
			}
		}
		kept = append(kept, frame)
	}

	out := make([]Entry, 0, len(kept))
	for _, frame := range kept {
		out = append(out, Entry{
			Category:    frame.cat,
			Path:        shortenHome(a.class.logical(frame.path), a.cfg.Home),
			Bytes:       frame.size,
			Reclaimable: frame.reclaimable,
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
