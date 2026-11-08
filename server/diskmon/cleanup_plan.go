package diskmon

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/xhd2015/dot-pkgs/go-pkgs/disk/reclaim"
)

const (
	// cleanupPlanTTL is how long measured sizes stay fresh. A stale plan is
	// still served while a background pass refreshes it, so the panel never
	// blocks on a walk over tens of gigabytes.
	cleanupPlanTTL = 10 * time.Minute
	// cleanupMeasureTimeout bounds an on-demand measurement (the system
	// classes). `xcrun simctl runtime list` can hang outside a GUI session, so a
	// stuck probe must never pin an HTTP request forever.
	cleanupMeasureTimeout = 90 * time.Second
)

// errCleanupMeasureTimeout is returned when an on-demand measurement overran
// cleanupMeasureTimeout; the probe keeps running and its result is cached.
var errCleanupMeasureTimeout = errors.New("measurement timed out")

var cleanupGroupOrder = []CleanupGroup{
	{ID: cleanupGroupDefault, Label: "Safe to clear", Note: "Recreated on demand; nothing here needs a reinstall."},
	{ID: cleanupGroupAggressive, Label: "Aggressive", Note: "Caches that will be re-downloaded or rebuilt: expect a slower next build or first run."},
	{ID: cleanupGroupSystem, Label: "System owned", Note: "Held by macOS. Clearing needs an administrator password and SIP may still refuse."},
	{ID: cleanupGroupInventory, Label: "Never cleared", Note: "Listed for transparency only: re-fetching these costs far more than the space."},
}

// cleanupService owns the measured plan shared by the report panel: one
// skeleton, one background measurement, one run at a time.
//
// Sizing every cache walks tens of gigabytes, so Plan never waits for it: the
// class list is returned immediately with measured=false rows and the page
// polls until pending clears.
type cleanupService struct {
	home func() (string, error)
	scan func(reclaim.Options, []string) (*reclaim.Plan, error)
	run  func(reclaim.Options) (*reclaim.Result, error)

	mu         sync.Mutex
	classes    []CleanupClass // skeleton: labels, actions, paths; no sizes
	sizes      map[string]*int64
	notes      map[string][]string
	details    map[string][]string
	volume     *CleanupVolume
	measuredAt time.Time
	scanMillis int64
	measuring  bool
	// gen invalidates an in-flight measurement after a run freed space.
	gen int

	runMu   sync.Mutex
	running bool
}

func newCleanupService() *cleanupService {
	return &cleanupService{
		home:    os.UserHomeDir,
		scan:    reclaim.ScanSteps,
		run:     reclaim.Run,
		sizes:   map[string]*int64{},
		notes:   map[string][]string{},
		details: map[string][]string{},
	}
}

// Plan returns the cleanup classes. Sizes come from the last completed
// measurement; when that is missing or older than cleanupPlanTTL a background
// pass starts and the response carries pending=true until it lands.
func (s *cleanupService) Plan(force bool) (*CleanupPlan, error) {
	home, err := s.home()
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.classes == nil {
		s.classes = cleanupClasses(home)
	}
	stale := s.measuredAt.IsZero() || time.Since(s.measuredAt) > cleanupPlanTTL
	if (force || stale) && !s.measuring {
		s.startMeasureLocked(home)
	}
	return s.assembleLocked(home), nil
}

// assembleLocked builds the wire plan from the skeleton plus known sizes.
func (s *cleanupService) assembleLocked(home string) *CleanupPlan {
	plan := &CleanupPlan{
		Home:       home,
		Volume:     s.volume,
		Groups:     cleanupGroupOrder,
		ScanMillis: s.scanMillis,
		Pending:    s.measuring || s.measuredAt.IsZero(),
	}
	if !s.measuredAt.IsZero() {
		plan.ScannedAt = s.measuredAt.Format(time.RFC3339)
	}
	for _, class := range s.classes {
		if size, ok := s.sizes[class.ID]; ok {
			class.SizeBytes = size
			class.Measured = size != nil
		}
		// Engine notes are Chinese diagnostics; keep them in Details so the
		// English UI never shows them, while the text stays available for
		// debugging and for the CLI.
		class.Details = append(class.Details, s.notes[class.ID]...)
		if d, ok := s.details[class.ID]; ok {
			class.Details = append(class.Details, d...)
		}
		accumulate(&plan.Totals, class)
		plan.Classes = append(plan.Classes, class)
	}
	return plan
}

// startMeasureLocked launches one measurement pass. Callers hold s.mu.
func (s *cleanupService) startMeasureLocked(home string) {
	s.measuring = true
	gen := s.gen
	go func() {
		plan, err := s.measureAll(home)
		s.mu.Lock()
		defer s.mu.Unlock()
		s.measuring = false
		if gen != s.gen || err != nil {
			// A run invalidated the data while we were walking, or the scan
			// failed: keep the old sizes and let the next Plan retry.
			return
		}
		s.mergeLocked(plan)
		s.measuredAt = time.Now()
		s.scanMillis = plan.ScanMillis
	}()
}

// measureAll sizes every class except the system tail, which shells out to
// xcrun / `log show` and is only measured on explicit request.
func (s *cleanupService) measureAll(home string) (*reclaim.Plan, error) {
	var ids []string
	for _, class := range s.classes {
		if class.Group != cleanupGroupSystem {
			ids = append(ids, class.ID)
		}
	}
	opts := reclaim.Options{Home: home, Stdout: io.Discard, Stderr: io.Discard}
	return s.scan(opts, ids)
}

// mergeLocked folds a scan result into the size caches.
func (s *cleanupService) mergeLocked(plan *reclaim.Plan) {
	for _, st := range plan.Steps {
		s.sizes[st.ID] = st.SizeBytes
		if len(st.Notes) > 0 {
			s.notes[st.ID] = st.Notes
		}
		if len(st.Details) > 0 {
			s.details[st.ID] = st.Details
		}
	}
	if v := plan.Volume; v != nil {
		s.volume = &CleanupVolume{Path: v.Path, TotalBytes: v.TotalBytes, FreeBytes: v.FreeBytes}
	}
}

// invalidate drops the measured sizes after a run changed the disk. The
// skeleton survives: labels and commands do not depend on disk state.
func (s *cleanupService) invalidate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gen++
	s.sizes = map[string]*int64{}
	s.notes = map[string][]string{}
	s.details = map[string][]string{}
	s.measuredAt = time.Time{}
	s.scanMillis = 0
}

// cleanupClasses builds the size-less skeleton from the shared registry.
func cleanupClasses(home string) []CleanupClass {
	infos := reclaim.StepInfos(home)
	classes := make([]CleanupClass, 0, len(infos))
	for _, info := range infos {
		classes = append(classes, cleanupClass(home, info))
	}
	return classes
}

// cleanupClass projects registry metadata into the wire shape.
func cleanupClass(home string, info reclaim.StepInfo) CleanupClass {
	return CleanupClass{
		ID:         info.ID,
		Group:      info.Group,
		Label:      cleanupLabel(info),
		Selected:   info.Group == cleanupGroupDefault && !info.ReadOnly,
		NeedsAdmin: info.NeedsAdmin,
		ReadOnly:   info.ReadOnly,
		Paths:      info.Paths,
		Actions:    cleanupCommands(home, info),
		Command:    cleanupHeadline(home, info),
		Notes:      cleanupNotes(info),
	}
}

func accumulate(t *CleanupTotals, class CleanupClass) {
	if class.SizeBytes == nil {
		t.Unmeasured++
		return
	}
	n := *class.SizeBytes
	switch class.Group {
	case cleanupGroupDefault:
		t.SelectedBytes += n
	case cleanupGroupAggressive:
		t.AggressiveBytes += n
	case cleanupGroupSystem:
		t.SystemBytes += n
	case cleanupGroupInventory:
		t.InventoryBytes += n
	}
}

// measureClass measures the given classes on demand (the system tail) and
// caches the result, so the next /plan shows their sizes. Read-only inventory
// rows are rejected: they are already measured with the rest of the skeleton.
func (s *cleanupService) measureClass(ids []string) error {
	home, err := s.home()
	if err != nil {
		return err
	}
	known := map[string]reclaim.StepInfo{}
	for _, in := range reclaim.StepInfos(home) {
		known[in.ID] = in
	}
	seen := map[string]bool{}
	unique := make([]string, 0, len(ids))
	for _, id := range ids {
		info, ok := known[id]
		if !ok {
			return fmt.Errorf("unknown cleanup class %q", id)
		}
		if info.ReadOnly {
			return fmt.Errorf("cleanup class %q is read-only; nothing to measure", id)
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		unique = append(unique, id)
	}
	if len(unique) == 0 {
		return errors.New("no cleanup class ids")
	}

	opts := reclaim.Options{Home: home, Stdout: io.Discard, Stderr: io.Discard}
	done := make(chan struct{})
	var scanned *reclaim.Plan
	var scanErr error
	go func() {
		defer close(done)
		scanned, scanErr = s.scan(opts, unique)
	}()
	timer := time.NewTimer(cleanupMeasureTimeout)
	defer timer.Stop()
	select {
	case <-done:
		if scanErr != nil {
			return scanErr
		}
		if scanned == nil {
			return errors.New("measurement returned no plan")
		}
		s.storeMeasured(scanned)
		return nil
	case <-timer.C:
		// Keep the probe running: when it finishes its numbers still land in
		// the cache, so the user's next refresh shows them.
		go func() {
			<-done
			if scanErr == nil && scanned != nil {
				s.storeMeasured(scanned)
			}
		}()
		return errCleanupMeasureTimeout
	}
}

func (s *cleanupService) storeMeasured(plan *reclaim.Plan) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mergeLocked(plan)
}

// tryStart guards against two concurrent runs: clearing twice at once would
// race the same paths and double-count the freed bytes.
func (s *cleanupService) tryStart() bool {
	s.runMu.Lock()
	defer s.runMu.Unlock()
	if s.running {
		return false
	}
	s.running = true
	return true
}

func (s *cleanupService) finish() {
	s.runMu.Lock()
	s.running = false
	s.runMu.Unlock()
	s.invalidate()
}

// sudoAlreadyGranted reports whether `sudo -n true` succeeds, i.e. no password
// dialog will appear for the system classes.
func sudoAlreadyGranted() bool {
	return exec.Command("sudo", "-n", "true").Run() == nil
}
