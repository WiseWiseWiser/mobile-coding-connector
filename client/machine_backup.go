package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// MachineBackupDirStat mirrors server dir rollup entries.
type MachineBackupDirStat struct {
	Path     string `json:"path"`
	Files    int    `json:"files"`
	Dirs     int    `json:"dirs"`
	Symlinks int    `json:"symlinks"`
}

// MachineBackupPlan is the JSON plan returned when dry_run is true.
type MachineBackupPlan struct {
	Home     string                 `json:"home"`
	DotFiles []string               `json:"dot_files"`
	DirStats []MachineBackupDirStat `json:"dir_stats"`
	Excluded []string               `json:"excluded"`
	Included []string               `json:"included"`
}

// MachineRestoreEntry mirrors one restore action.
type MachineRestoreEntry struct {
	Path   string `json:"path"`
	Action string `json:"action"`
}

// MachineRestorePlan is returned for restore dry-run and apply.
type MachineRestorePlan struct {
	Home    string                `json:"home"`
	Entries []MachineRestoreEntry `json:"entries"`
}

type machineBackupRequestBody struct {
	DryRun  bool     `json:"dry_run"`
	Exclude []string `json:"exclude"`
	Include []string `json:"include"`
}

func (c *Client) machineBackupBody(exclude, include []string, dryRun bool) machineBackupRequestBody {
	if exclude == nil {
		exclude = []string{}
	}
	if include == nil {
		include = []string{}
	}
	return machineBackupRequestBody{DryRun: dryRun, Exclude: exclude, Include: include}
}

// MachineBackupPlan calls backup with dry_run=true.
func (c *Client) MachineBackupPlan(exclude, include []string) (*MachineBackupPlan, error) {
	var out MachineBackupPlan
	if err := c.postJSON("/api/remote-agent/machine/backup", c.machineBackupBody(exclude, include, true), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// MachineBackupArchive calls backup with dry_run=false and returns the tar.xz body.
func (c *Client) MachineBackupArchive(exclude, include []string) (io.ReadCloser, error) {
	data, err := json.Marshal(c.machineBackupBody(exclude, include, false))
	if err != nil {
		return nil, fmt.Errorf("marshal backup request: %w", err)
	}
	httpReq, err := c.NewRequest(http.MethodPost, "/api/remote-agent/machine/backup", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(httpReq)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		return nil, readAPIError(resp)
	}
	return resp.Body, nil
}

// MachineRestorePlan uploads an archive with dry_run=true.
func (c *Client) MachineRestorePlan(archive io.Reader, exclude, include []string) (*MachineRestorePlan, error) {
	return c.machineRestore(archive, true, exclude, include)
}

// MachineRestoreApply uploads an archive with dry_run=false.
func (c *Client) MachineRestoreApply(archive io.Reader, exclude, include []string) (*MachineRestorePlan, error) {
	return c.machineRestore(archive, false, exclude, include)
}

func (c *Client) machineRestore(archive io.Reader, dryRun bool, exclude, include []string) (*MachineRestorePlan, error) {
	path := "/api/remote-agent/machine/restore"
	query := url.Values{}
	if dryRun {
		query.Set("dry_run", "true")
	}
	for _, ex := range exclude {
		query.Add("exclude", ex)
	}
	for _, inc := range include {
		query.Add("include", inc)
	}
	if len(query) > 0 {
		path += "?" + query.Encode()
	}
	httpReq, err := c.NewRequest(http.MethodPost, path, archive)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/x-xz")
	resp, err := c.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, readAPIError(resp)
	}
	var out MachineRestorePlan
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode restore plan: %w", err)
	}
	return &out, nil
}