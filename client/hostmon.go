package client

import (
	"net/url"
	"strconv"
)

// HostMonProc is one process in a host-monitor sample.
type HostMonProc struct {
	PID      int     `json:"pid"`
	Comm     string  `json:"comm"`
	Cmd      string  `json:"cmd,omitempty"`
	ReadBps  float64 `json:"readBps"`
	WriteBps float64 `json:"writeBps"`
	CPU      float64 `json:"cpu"`
}

// HostMonSample is one persisted CPU and disk observation.
type HostMonSample struct {
	T     string  `json:"t"`
	Load1 float64 `json:"load1"`
	CPU   struct {
		User   float64 `json:"user"`
		System float64 `json:"system"`
		IOWait float64 `json:"iowait"`
		Idle   float64 `json:"idle"`
	} `json:"cpu"`
	Disk struct {
		Dev   string  `json:"dev"`
		RIops float64 `json:"riops"`
		WIops float64 `json:"wiops"`
		RBps  float64 `json:"rbps"`
		WBps  float64 `json:"wbps"`
	} `json:"disk"`
	Top []HostMonProc `json:"top,omitempty"`
}

// HostMonStatus is the sampler control state.
type HostMonStatus struct {
	Running     bool           `json:"running"`
	AutoStart   bool           `json:"autoStart"`
	IntervalSec int            `json:"intervalSec"`
	LogPath     string         `json:"logPath"`
	LogBytes    int64          `json:"logBytes"`
	Latest      *HostMonSample `json:"latest,omitempty"`
}

// HostMonStatus fetches the sampler state and the newest sample.
func (c *Client) HostMonStatus() (*HostMonStatus, error) {
	var out HostMonStatus
	if err := c.getJSON("/api/remote-agent/hostmon/status", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// HostMonSamples fetches the last n persisted samples, oldest first.
func (c *Client) HostMonSamples(n int) ([]HostMonSample, error) {
	path := "/api/remote-agent/hostmon/samples?lines=" + url.QueryEscape(strconv.Itoa(n))
	var out struct {
		Samples []HostMonSample `json:"samples"`
	}
	if err := c.getJSON(path, &out); err != nil {
		return nil, err
	}
	if out.Samples == nil {
		out.Samples = []HostMonSample{}
	}
	return out.Samples, nil
}

// HostMonControl posts start, stop, enable, or disable.
func (c *Client) HostMonControl(action string) (*HostMonStatus, error) {
	var out HostMonStatus
	if err := c.postJSON("/api/remote-agent/hostmon/"+action, map[string]string{}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
