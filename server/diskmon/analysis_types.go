package diskmon

import "time"

// AnalysisSchema versions the category rules. Trends only compare reports that
// share a schema, so adding or renaming a category never fabricates a jump.
const AnalysisSchema = 1

// Category is one bucket of the space breakdown. ID is the stable join key
// used by the trend charts; Label is free to change.
type Category struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Kind        string   `json:"kind"` // cache | media | doc | other
	Bytes       int64    `json:"bytes"`
	Reclaimable int64    `json:"reclaimable"`
	Paths       []string `json:"paths,omitempty"`
}

// Entry is one large directory found during a scan.
type Entry struct {
	Category    string `json:"category"`
	Path        string `json:"path"`
	Bytes       int64  `json:"bytes"`
	Reclaimable bool   `json:"reclaimable"`
}

// Report is one completed scan. Reports are append-only; older ones lose their
// per-directory entries but keep the category rollup that trends need.
type Report struct {
	ID             string     `json:"id"`
	Schema         int        `json:"schema"`
	ScannedAt      time.Time  `json:"scanned_at"`
	ScanMillis     int64      `json:"scan_ms"`
	Complete       bool       `json:"complete"`
	Truncated      bool       `json:"truncated"`
	UsedBytes      int64      `json:"used_bytes"`
	TotalBytes     int64      `json:"total_bytes"`
	ScannedBytes   int64      `json:"scanned_bytes"`
	Files          int64      `json:"files"`
	NoAccessDirs   int        `json:"no_access_dirs"`
	Roots          []string   `json:"roots,omitempty"`
	Categories     []Category `json:"categories"`
	Entries        []Entry    `json:"entries,omitempty"`
	EntriesDropped bool       `json:"entries_dropped,omitempty"`
}

// ReportSummary is the list/trend shape: everything except the directory rows.
type ReportSummary struct {
	ID             string     `json:"id"`
	Schema         int        `json:"schema"`
	ScannedAt      time.Time  `json:"scanned_at"`
	ScanMillis     int64      `json:"scan_ms"`
	Complete       bool       `json:"complete"`
	Truncated      bool       `json:"truncated"`
	UsedBytes      int64      `json:"used_bytes"`
	TotalBytes     int64      `json:"total_bytes"`
	ScannedBytes   int64      `json:"scanned_bytes"`
	Files          int64      `json:"files"`
	NoAccessDirs   int        `json:"no_access_dirs"`
	Reclaimable    int64      `json:"reclaimable"`
	HasEntries     bool       `json:"has_entries"`
	Categories     []Category `json:"categories"`
	EntriesDropped bool       `json:"entries_dropped,omitempty"`
}

// JobProgress is what the running scan publishes. Percent is a heuristic: the
// scan cannot know the denominator before it finishes.
type JobProgress struct {
	ID           string    `json:"id"`
	StartedAt    time.Time `json:"started_at"`
	Phase        string    `json:"phase"`
	WalkedBytes  int64     `json:"walked_bytes"`
	Files        int64     `json:"files"`
	NoAccessDirs int       `json:"no_access_dirs"`
	Percent      int       `json:"percent"`
	State        string    `json:"state"` // running | cancelled | failed
}

// Attempt records how the newest try ended, independent of whether it produced
// a report. An interrupted try must never look like a completed scan.
type Attempt struct {
	At      time.Time `json:"at"`
	Outcome string    `json:"outcome"` // completed | cancelled | failed | interrupted
	JobID   string    `json:"job_id,omitempty"`
	Error   string    `json:"error,omitempty"`
}

// AnalysisStatus is the single source of truth the page polls.
type AnalysisStatus struct {
	Running         bool           `json:"running"`
	Job             *JobProgress   `json:"job,omitempty"`
	LastCompletedAt *time.Time     `json:"last_completed_at,omitempty"`
	LastAttempt     *Attempt       `json:"last_attempt,omitempty"`
	ReportCount     int            `json:"report_count"`
	Latest          *ReportSummary `json:"latest,omitempty"`
}

func (r Report) summary() ReportSummary {
	return ReportSummary{
		ID:             r.ID,
		Schema:         r.Schema,
		ScannedAt:      r.ScannedAt,
		ScanMillis:     r.ScanMillis,
		Complete:       r.Complete,
		Truncated:      r.Truncated,
		UsedBytes:      r.UsedBytes,
		TotalBytes:     r.TotalBytes,
		ScannedBytes:   r.ScannedBytes,
		Files:          r.Files,
		NoAccessDirs:   r.NoAccessDirs,
		Reclaimable:    r.reclaimableTotal(),
		HasEntries:     len(r.Entries) > 0,
		Categories:     r.Categories,
		EntriesDropped: r.EntriesDropped,
	}
}

func (r Report) reclaimableTotal() int64 {
	var sum int64
	for _, c := range r.Categories {
		sum += c.Reclaimable
	}
	return sum
}

// progressPercent keeps the bar honest: it never claims 100% before the report
// exists, and it is explicitly an estimate while walking.
func progressPercent(walked, used int64) int {
	if used <= 0 || walked <= 0 {
		return 0
	}
	budget := used * 9 / 10
	if budget <= 0 {
		return 0
	}
	pct := int(walked * 100 / budget)
	if pct > 99 {
		return 99
	}
	if pct < 0 {
		return 0
	}
	return pct
}
