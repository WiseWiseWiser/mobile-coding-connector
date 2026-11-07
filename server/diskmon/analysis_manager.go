package diskmon

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"
)

// scanBudget is a soft cap: a truncated report still carries the categories
// found so far and says so, which beats losing the whole run. A 9M-file /
// 600 GB developer volume needs about three minutes of parallel walking plus a
// second per unreadable directory, so the cap leaves room for a complete report
// on a real machine while still cutting off a runaway one.
const scanBudget = 300 * time.Second

var errAnalysisRunning = errors.New("an analysis is already running")

// runningAnalysis is the one global scan. Its progress is persisted every couple
// of seconds so a page opened later still sees where the walk is.
type runningAnalysis struct {
	progress  JobProgress
	cancel    context.CancelFunc
	persisted time.Time
}

type analysisManager struct {
	store *analysisStore
	now   func() time.Time
	scan  func(ctx context.Context, cfg scanConfig) (scanResult, error)
	usage func() (used, total int64)
	roots func() []string

	mu  sync.Mutex
	job *runningAnalysis
}

func newAnalysisManager(store *analysisStore) *analysisManager {
	return &analysisManager{
		store: store,
		now:   time.Now,
		scan:  runScan,
		usage: volumeUsage,
		roots: defaultScanRoots,
	}
}

// volumeUsage anchors the progress estimate to the primary volume.
func volumeUsage() (used, total int64) {
	snap, err := Get()
	if err != nil || snap == nil {
		return 0, 0
	}
	for _, volume := range snap.Volumes {
		if volume.Primary {
			return volume.UsedBytes, volume.TotalBytes
		}
	}
	if len(snap.Volumes) > 0 {
		return snap.Volumes[0].UsedBytes, snap.Volumes[0].TotalBytes
	}
	return 0, 0
}

// Start launches a scan, or reports the running one so a second click attaches
// instead of queueing duplicate work.
func (m *analysisManager) Start() (JobProgress, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.job != nil {
		return m.job.progress, errAnalysisRunning
	}

	now := m.now()
	ctx, cancel := context.WithCancel(context.Background())
	job := &runningAnalysis{
		progress: JobProgress{
			ID:        strconv.FormatInt(now.UnixNano(), 36),
			StartedAt: now,
			Phase:     "preparing",
			State:     "running",
		},
		cancel: cancel,
	}
	m.job = job
	m.store.writeJob(job.progress)
	job.persisted = now

	go m.run(ctx, job)
	return job.progress, nil
}

func (m *analysisManager) run(ctx context.Context, job *runningAnalysis) {
	used, total := m.usage()
	roots := m.roots()

	startedAt := time.Now()
	result, err := m.scan(ctx, scanConfig{
		Roots:  roots,
		Budget: scanBudget,
		Progress: func(walked, files int64, noAccess int, phase string) {
			m.noteProgress(job, walked, files, noAccess, phase, used)
		},
	})
	scanMillis := time.Since(startedAt).Milliseconds()

	switch {
	case errors.Is(err, context.Canceled):
		// Cancelling never writes a report: a partial scan must not masquerade
		// as a finished one in the history.
		m.finish(job, Attempt{At: m.now(), Outcome: "cancelled", JobID: job.progress.ID})
	case err != nil:
		m.finish(job, Attempt{At: m.now(), Outcome: "failed", JobID: job.progress.ID, Error: err.Error()})
	default:
		report := Report{
			ID:           m.store.nextReportID(job.progress.StartedAt),
			Schema:       AnalysisSchema,
			ScannedAt:    job.progress.StartedAt,
			ScanMillis:   scanMillis,
			Complete:     !result.Truncated,
			Truncated:    result.Truncated,
			UsedBytes:    used,
			TotalBytes:   total,
			ScannedBytes: result.WalkedBytes,
			Files:        result.Files,
			NoAccessDirs: result.NoAccessDirs,
			Roots:        result.Roots,
			Categories:   result.Categories,
			Entries:      result.Entries,
			Coverage:     scanCoverage(result),
		}
		if embedErr := m.store.append(report, job.progress.ID); embedErr != nil {
			m.finish(job, Attempt{
				At:      m.now(),
				Outcome: "failed",
				JobID:   job.progress.ID,
				Error:   "the report could not be saved: " + embedErr.Error(),
			})
			return
		}
		m.finish(job, Attempt{At: m.now(), Outcome: "completed", JobID: job.progress.ID})
	}
}

// scanCoverage turns the walk's coverage counters into the shape the page reads.
// The priority set is the part the headline numbers promise, so its completion
// is reported separately from a whole-volume truncation.
func scanCoverage(result scanResult) *ScanCoverage {
	coverage := &ScanCoverage{
		PriorityRoots:    result.PriorityRoots,
		PriorityScanned:  result.PriorityScanned,
		PriorityComplete: len(result.PriorityRoots) > 0 && result.PriorityScanned >= len(result.PriorityRoots),
		Skipped:          result.Skipped,
	}
	if len(coverage.PriorityRoots) == 0 && len(coverage.Skipped) == 0 {
		return nil
	}
	return coverage
}

// finish releases the running slot before recording the attempt, so a retry can
// start immediately even if the metadata write is slow.
func (m *analysisManager) finish(job *runningAnalysis, attempt Attempt) {
	m.mu.Lock()
	if m.job == job {
		m.job = nil
	}
	m.mu.Unlock()

	job.cancel()
	m.store.recordAttempt(attempt)
}

func (m *analysisManager) noteProgress(job *runningAnalysis, walked, files int64, noAccess int, phase string, used int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.job != job {
		return
	}
	job.progress.WalkedBytes = walked
	job.progress.Files = files
	job.progress.NoAccessDirs = noAccess
	job.progress.Percent = progressPercent(walked, used)
	if phase != "" {
		job.progress.Phase = phase
	}
	now := m.now()
	if now.Sub(job.persisted) >= 2*time.Second {
		job.persisted = now
		m.store.writeJob(job.progress)
	}
}

// Cancel asks the walk to stop. It is idempotent and false when nothing runs.
func (m *analysisManager) Cancel() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.job == nil {
		return false
	}
	m.job.cancel()
	return true
}

func (m *analysisManager) Status() AnalysisStatus {
	m.mu.Lock()
	var job *JobProgress
	if m.job != nil {
		current := m.job.progress
		job = &current
	}
	m.mu.Unlock()

	meta, summaries, count := m.store.snapshot(1)
	status := AnalysisStatus{
		Running:         job != nil,
		Job:             job,
		LastCompletedAt: meta.LastCompletedAt,
		LastAttempt:     meta.LastAttempt,
		ReportCount:     count,
	}
	if len(summaries) > 0 {
		latest := summaries[0]
		status.Latest = &latest
	}
	return status
}

func (m *analysisManager) Reports(limit int) (analysisMeta, []ReportSummary, int) {
	return m.store.snapshot(limit)
}

func (m *analysisManager) Report(id string) (Report, bool) {
	return m.store.report(id)
}
