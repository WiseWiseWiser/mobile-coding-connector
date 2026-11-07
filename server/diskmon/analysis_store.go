package diskmon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"
)

const (
	analysisStoreVersion = 1
	// analysisMaxReports caps the history. Each report is a few KB once the
	// per-directory rows are dropped, so 50 stays well under a megabyte.
	analysisMaxReports = 50
	// analysisKeepEntries keeps directory rows for the newest reports only.
	analysisKeepEntries = 5
)

type analysisFile struct {
	Version int      `json:"version"`
	Reports []Report `json:"reports"`
}

type analysisMeta struct {
	LastCompletedAt *time.Time `json:"last_completed_at,omitempty"`
	LastAttempt     *Attempt   `json:"last_attempt,omitempty"`
}

// analysisStore owns the on-disk history: one JSON file for reports, one for
// metadata, one for the job that is running right now.
type analysisStore struct {
	dir string

	mu      sync.Mutex
	loaded  bool
	reports []Report
	meta    analysisMeta
}

func newAnalysisStore(dir string) *analysisStore {
	return &analysisStore{dir: dir}
}

// defaultAnalysisDir follows the ~/.ai-critic convention used by the uploader.
func defaultAnalysisDir() string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".ai-critic", "disk")
	}
	return filepath.Join(".ai-critic", "disk")
}

func (s *analysisStore) path(name string) string {
	return filepath.Join(s.dir, name)
}

func (s *analysisStore) loadLocked() {
	if s.loaded {
		return
	}
	s.loaded = true
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return
	}

	if data, err := os.ReadFile(s.path("analysis.json")); err == nil {
		var file analysisFile
		if json.Unmarshal(data, &file) != nil {
			// Broken history is moved aside, never silently overwritten.
			_ = os.Rename(s.path("analysis.json"), s.path("analysis.json.corrupt-"+time.Now().Format("20060102-150405")))
		} else {
			for _, report := range file.Reports {
				if report.ID == "" || report.ScannedAt.IsZero() {
					continue
				}
				if report.Schema == 0 {
					report.Schema = AnalysisSchema
				}
				s.reports = append(s.reports, report)
			}
		}
	}

	if data, err := os.ReadFile(s.path("meta.json")); err == nil {
		var meta analysisMeta
		if json.Unmarshal(data, &meta) == nil {
			s.meta = meta
		}
	}

	// A job file left behind means the previous process died mid-scan. That try
	// must never be presented as a completed report.
	if data, err := os.ReadFile(s.path("job.json")); err == nil {
		var job JobProgress
		if json.Unmarshal(data, &job) == nil && job.ID != "" {
			at := job.StartedAt
			if at.IsZero() {
				at = time.Now()
			}
			s.meta.LastAttempt = &Attempt{
				At:      at,
				Outcome: "interrupted",
				JobID:   job.ID,
				Error:   "the server stopped before the scan finished",
			}
			s.writeMetaLocked()
		}
		_ = os.Remove(s.path("job.json"))
	}

	sort.SliceStable(s.reports, func(i, j int) bool {
		return s.reports[i].ScannedAt.After(s.reports[j].ScannedAt)
	})
	if len(s.reports) > analysisMaxReports {
		s.reports = s.reports[:analysisMaxReports]
	}
	s.dropEntriesLocked()
}

// dropEntriesLocked keeps directory rows only for the newest reports; the
// category rollup that trends need is always kept.
func (s *analysisStore) dropEntriesLocked() {
	for i := analysisKeepEntries; i < len(s.reports); i++ {
		if len(s.reports[i].Entries) > 0 {
			s.reports[i].Entries = nil
			s.reports[i].EntriesDropped = true
		}
	}
}

func (s *analysisStore) writeReportsLocked() error {
	file := analysisFile{Version: analysisStoreVersion, Reports: s.reports}
	return writeJSONFile(s.path("analysis.json"), file)
}

func (s *analysisStore) writeMetaLocked() {
	_ = writeJSONFile(s.path("meta.json"), s.meta)
}

// append stores a finished report as the newest entry.
func (s *analysisStore) append(report Report, jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	if report.Schema == 0 {
		report.Schema = AnalysisSchema
	}
	s.reports = append([]Report{report}, s.reports...)
	if len(s.reports) > analysisMaxReports {
		s.reports = s.reports[:analysisMaxReports]
	}
	s.dropEntriesLocked()
	if err := s.writeReportsLocked(); err != nil {
		return err
	}
	at := report.ScannedAt
	s.meta.LastCompletedAt = &at
	s.meta.LastAttempt = &Attempt{At: time.Now(), Outcome: "completed", JobID: jobID}
	s.writeMetaLocked()
	return nil
}

// recordAttempt notes how a try ended when it produced no report.
func (s *analysisStore) recordAttempt(attempt Attempt) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	s.meta.LastAttempt = &attempt
	s.writeMetaLocked()
	_ = os.Remove(s.path("job.json"))
}

func (s *analysisStore) writeJob(job JobProgress) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = writeJSONFile(s.path("job.json"), job)
}

func (s *analysisStore) clearJob() {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = os.Remove(s.path("job.json"))
}

func (s *analysisStore) snapshot(limit int) (analysisMeta, []ReportSummary, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	count := len(s.reports)
	if limit <= 0 || limit > count {
		limit = count
	}
	summaries := make([]ReportSummary, 0, limit)
	for _, report := range s.reports[:limit] {
		summaries = append(summaries, report.summary())
	}
	meta := s.meta
	return meta, summaries, count
}

func (s *analysisStore) report(id string) (Report, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	for _, report := range s.reports {
		if report.ID == id {
			return report, true
		}
	}
	return Report{}, false
}

// nextReportID builds a sortable, human-readable id from the start time, and
// disambiguates scans that started inside the same second.
func (s *analysisStore) nextReportID(at time.Time) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	base := at.Local().Format("20060102-150405")
	id := base
	for suffix := 2; ; suffix++ {
		taken := false
		for _, report := range s.reports {
			if report.ID == id {
				taken = true
				break
			}
		}
		if !taken {
			return id
		}
		id = base + "-" + strconv.Itoa(suffix)
	}
}

func writeJSONFile(path string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
