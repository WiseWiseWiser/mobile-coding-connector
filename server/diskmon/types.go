package diskmon

// Volume is one mounted filesystem whose capacity can be reported.
type Volume struct {
	Mount       string  `json:"mount"`
	Device      string  `json:"device"`
	FSType      string  `json:"fs_type"`
	TotalBytes  int64   `json:"total_bytes"`
	UsedBytes   int64   `json:"used_bytes"`
	FreeBytes   int64   `json:"free_bytes"`
	UsedPercent float64 `json:"used_percent"`
	ReadOnly    bool    `json:"read_only"`
	Primary     bool    `json:"primary"`
}

// MergedMount documents a mount that folds into another volume. It is not
// counted in Volumes, so the same APFS container never shows up twice.
type MergedMount struct {
	Mount  string `json:"mount"`
	Device string `json:"device"`
	FSType string `json:"fs_type"`
	Reason string `json:"reason"`
}

// IO mirrors the newest hostmon disk sample. Field timestamps stay strings so
// the JSON matches hostmon's own logs.
type IO struct {
	Device      string  `json:"device"`
	ReadBps     float64 `json:"read_bps"`
	WriteBps    float64 `json:"write_bps"`
	ReadIOPS    float64 `json:"read_iops"`
	WriteIOPS   float64 `json:"write_iops"`
	IntervalSec int     `json:"interval_sec"`
	SampledAt   string  `json:"sampled_at"`
}

// Snapshot is the JSON body of GET /api/server/disk.
type Snapshot struct {
	SampledAt    string        `json:"sampled_at"`
	PrimaryMount string        `json:"primary_mount"`
	Volumes      []Volume      `json:"volumes"`
	Merged       []MergedMount `json:"merged,omitempty"`
	IO           *IO           `json:"io,omitempty"`
	Warning      string        `json:"warning,omitempty"`
}
