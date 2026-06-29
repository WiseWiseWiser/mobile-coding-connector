package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// PullLocalRequest is sent to POST /api/remote-agent/project/pull-local.
type PullLocalRequest struct {
	Dir          string
	DryRun       bool
	IncludeFiles []string
	MaxSizeBytes int64
}

// PullLocalOversizedFile mirrors server oversized file entries in a dry-run plan.
type PullLocalOversizedFile struct {
	Path     string `json:"path"`
	Bytes    int64  `json:"bytes"`
	Included bool   `json:"included"`
}

// PullLocalPlan is the JSON plan returned when DryRun is true.
type PullLocalPlan struct {
	Dir             string                   `json:"dir"`
	Commit          string                   `json:"commit"`
	Branch          string                   `json:"branch"`
	OriginURL       string                   `json:"origin_url"`
	IsClean         bool                     `json:"is_clean"`
	TrackedFiles    int                      `json:"tracked_files"`
	UntrackedFiles  int                      `json:"untracked_files"`
	DeletedFiles    int                      `json:"deleted_files"`
	SubmodulesOK    bool                     `json:"submodules_ok"`
	DirtySubmodules []string                 `json:"dirty_submodules"`
	EstimatedBytes  int64                    `json:"estimated_bytes"`
	OversizedFiles  []PullLocalOversizedFile `json:"oversized_files"`
	WithinMaxSize   bool                     `json:"within_max_size"`
}

type pullLocalRequestBody struct {
	Dir          string   `json:"dir"`
	DryRun       bool     `json:"dry_run"`
	IncludeFiles []string `json:"include_files"`
	MaxSizeBytes int64    `json:"max_size_bytes"`
}

func (c *Client) pullLocalBody(req PullLocalRequest) pullLocalRequestBody {
	include := req.IncludeFiles
	if include == nil {
		include = []string{}
	}
	return pullLocalRequestBody{
		Dir:          req.Dir,
		DryRun:       req.DryRun,
		IncludeFiles: include,
		MaxSizeBytes: req.MaxSizeBytes,
	}
}

// PullLocal calls pull-local with dry_run=true and decodes the plan JSON.
func (c *Client) PullLocal(req PullLocalRequest) (*PullLocalPlan, error) {
	req.DryRun = true
	var out PullLocalPlan
	if err := c.postJSON("/api/remote-agent/project/pull-local", c.pullLocalBody(req), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// PullLocalPackage calls pull-local with dry_run=false and returns the tar.gz body.
func (c *Client) PullLocalPackage(req PullLocalRequest) (io.ReadCloser, error) {
	req.DryRun = false
	data, err := json.Marshal(c.pullLocalBody(req))
	if err != nil {
		return nil, fmt.Errorf("marshal pull-local request: %w", err)
	}
	httpReq, err := c.NewRequest(http.MethodPost, "/api/remote-agent/project/pull-local", bytes.NewReader(data))
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

// PullLocalTruncate resets the remote repo to commit via the truncate API.
func (c *Client) PullLocalTruncate(dir, commit string) error {
	body := map[string]string{"dir": dir, "commit": commit}
	return c.postJSON("/api/remote-agent/project/pull-local/truncate", body, nil)
}