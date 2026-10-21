package hostmon

import "time"

const (
	// Interval is how often a running sampler records one line.
	Interval = 15 * time.Second
	// TopN is how many I/O processes each sample keeps.
	TopN = 5
	// CPUExtraPercent adds a process that is not already in the I/O top set.
	CPUExtraPercent = 20
	// CmdMax is the stored command line length.
	CmdMax = 80

	logName     = "samples.log"
	enabledName = "enabled"
)

// MaxLogBytes is the cap for samples.log before it rotates to samples.log.1.
var MaxLogBytes int64 = 8 << 20

// Sample is one persisted observation. Fields stay small so a fsync is cheap.
type Sample struct {
	T     string       `json:"t"`
	Load1 float64      `json:"load1"`
	CPU   CPUSample    `json:"cpu"`
	Disk  DiskSample   `json:"disk"`
	Top   []ProcSample `json:"top,omitempty"`
}

// CPUSample is percent of the interval, 0–100.
type CPUSample struct {
	User   float64 `json:"user"`
	System float64 `json:"system"`
	IOWait float64 `json:"iowait"`
	Idle   float64 `json:"idle"`
}

// DiskSample is the busiest whole-disk device over the interval.
type DiskSample struct {
	Dev   string  `json:"dev"`
	RIops float64 `json:"riops"`
	WIops float64 `json:"wiops"`
	RBps  float64 `json:"rbps"`
	WBps  float64 `json:"wbps"`
}

// ProcSample is one process. CPU is percent of one core.
type ProcSample struct {
	PID      int     `json:"pid"`
	Comm     string  `json:"comm"`
	Cmd      string  `json:"cmd,omitempty"`
	ReadBps  float64 `json:"readBps"`
	WriteBps float64 `json:"writeBps"`
	CPU      float64 `json:"cpu"`
}

// Status is the sampler's control state plus the newest sample, if any.
type Status struct {
	Running     bool    `json:"running"`
	AutoStart   bool    `json:"autoStart"`
	IntervalSec int     `json:"intervalSec"`
	LogPath     string  `json:"logPath"`
	LogBytes    int64   `json:"logBytes"`
	Latest      *Sample `json:"latest,omitempty"`
}
