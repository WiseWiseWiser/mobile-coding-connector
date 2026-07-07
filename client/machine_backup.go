package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/xhd2015/ai-critic/server/machinebackup"
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
	DryRun              bool     `json:"dry_run"`
	Exclude             []string `json:"exclude"`
	Include             []string `json:"include"`
	SkipGitDirsScan     bool     `json:"skip_git_dirs_scan,omitempty"`
	GitDirsScanMaxDepth int      `json:"git_dirs_scan_max_depth,omitempty"`
}

// MachineBackupEffectiveConfig returns the merged exclusion config from the server.
// Optional exclude/include/largeDirThreshold are forwarded as query params for CLI preview.
func (c *Client) MachineBackupEffectiveConfig(exclude, include []string, largeDirThreshold string) (*machinebackup.ExclusionConfig, error) {
	query := url.Values{}
	for _, ex := range exclude {
		query.Add("exclude", ex)
	}
	for _, inc := range include {
		query.Add("include", inc)
	}
	if strings.TrimSpace(largeDirThreshold) != "" {
		query.Set("large_dir_threshold", strings.TrimSpace(largeDirThreshold))
	}
	path := "/api/remote-agent/machine/backup-config"
	if len(query) > 0 {
		path += "?" + query.Encode()
	}
	var out machinebackup.ExclusionConfig
	if err := c.getJSON(path, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// MachineBackupSetConfig persists user exclude paths and optional threshold on the server.
func (c *Client) MachineBackupSetConfig(exclude []string, largeDirThreshold string) (*machinebackup.ExclusionConfig, error) {
	if exclude == nil {
		exclude = []string{}
	}
	body := map[string]any{"exclude": exclude}
	if strings.TrimSpace(largeDirThreshold) != "" {
		body["large_dir_threshold"] = strings.TrimSpace(largeDirThreshold)
	}
	var out machinebackup.ExclusionConfig
	if err := c.sendJSON(http.MethodPut, "/api/remote-agent/machine/backup-config", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type MachineBackupOptions struct {
	SkipGitDirsScan     bool
	GitDirsScanMaxDepth int
}

func (c *Client) machineBackupBody(exclude, include []string, dryRun bool, opts MachineBackupOptions) machineBackupRequestBody {
	if exclude == nil {
		exclude = []string{}
	}
	if include == nil {
		include = []string{}
	}
	return machineBackupRequestBody{
		DryRun:              dryRun,
		Exclude:             exclude,
		Include:             include,
		SkipGitDirsScan:     opts.SkipGitDirsScan,
		GitDirsScanMaxDepth: opts.GitDirsScanMaxDepth,
	}
}

// MachineBackupPlan calls backup with dry_run=true.
func (c *Client) MachineBackupPlan(exclude, include []string) (*MachineBackupPlan, error) {
	var out MachineBackupPlan
	if err := c.postJSON("/api/remote-agent/machine/backup", c.machineBackupBody(exclude, include, true, MachineBackupOptions{}), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// MachineBackupArchive calls backup with dry_run=false and returns the tar.xz body.
func (c *Client) MachineBackupArchive(exclude, include []string, opts MachineBackupOptions) (io.ReadCloser, error) {
	data, err := json.Marshal(c.machineBackupBody(exclude, include, false, opts))
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