package procmon

import "time"

const (
	minSampleInterval = time.Second
	firstDeltaWait    = 200 * time.Millisecond
)

// Snapshot is one live view of host resource use plus the process table.
type Snapshot struct {
	SampledAt  string    `json:"sampled_at"`
	IntervalMs int       `json:"interval_ms"`
	CPU        CPU       `json:"cpu"`
	Load       Load      `json:"load"`
	Memory     Memory    `json:"memory"`
	Counts     Counts    `json:"counts"`
	SelfPID    int       `json:"self_pid"`
	Processes  []Process `json:"processes"`
}

// CPU is host-wide usage as a percent of all cores (0–100).
type CPU struct {
	NumCPU      int     `json:"num_cpu"`
	UsedPercent float64 `json:"used_percent"`
}

// Load is 1/5/15 minute load averages.
type Load struct {
	M1  float64 `json:"m1"`
	M5  float64 `json:"m5"`
	M15 float64 `json:"m15"`
}

// Memory is RAM plus optional compressor/swap figures.
type Memory struct {
	Total           uint64  `json:"total"`
	Used            uint64  `json:"used"`
	Free            uint64  `json:"free"`
	UsedPercent     float64 `json:"used_percent"`
	CompressorBytes uint64  `json:"compressor_bytes,omitempty"`
	SwapTotal       uint64  `json:"swap_total,omitempty"`
	SwapUsed        uint64  `json:"swap_used,omitempty"`
	SwapFree        uint64  `json:"swap_free,omitempty"`
	SwapUsedPercent float64 `json:"swap_used_percent,omitempty"`
}

// Counts is process table size.
type Counts struct {
	Processes int `json:"processes"`
}

// Process is one row in the task manager.
type Process struct {
	PID        int      `json:"pid"`
	PPID       int      `json:"ppid"`
	Name       string   `json:"name"`
	Cmd        string   `json:"cmd"`
	User       string   `json:"user"`
	State      string   `json:"state"`
	CPUPercent float64  `json:"cpu_percent"`
	RSSBytes   uint64   `json:"rss_bytes"`
	ElapsedSec int64    `json:"elapsed_sec"`
	Protected  string   `json:"protected,omitempty"`
	Tags       []string `json:"tags,omitempty"`
}

type rawProc struct {
	PID     int
	PPID    int
	UID     int
	User    string
	State   string
	RSSKiB  uint64
	Elapsed time.Duration
	CPUTime time.Duration
	Cmd     string
}
