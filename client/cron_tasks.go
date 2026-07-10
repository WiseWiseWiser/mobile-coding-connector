package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// CronTaskDefinition is the create/update body for /api/cron-tasks.
type CronTaskDefinition struct {
	ID           string            `json:"id,omitempty"`
	Name         string            `json:"name"`
	Command      string            `json:"command"`
	WorkingDir   string            `json:"workingDir,omitempty"`
	ExtraEnv     map[string]string `json:"extraEnv,omitempty"`
	ScheduleMode string            `json:"scheduleMode,omitempty"`
	Interval     string            `json:"interval,omitempty"`
	CronExpr     string            `json:"cronExpr,omitempty"`
	Enabled      *bool             `json:"enabled,omitempty"`
	Timeout      string            `json:"timeout,omitempty"`
}

// CronTaskRun is one history entry.
type CronTaskRun struct {
	StartedAt  string `json:"startedAt"`
	FinishedAt string `json:"finishedAt,omitempty"`
	ExitCode   *int   `json:"exitCode,omitempty"`
	Error      string `json:"error,omitempty"`
}

// CronTaskStatus is returned by list/create/update/enable/disable/run.
type CronTaskStatus struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Command        string            `json:"command"`
	WorkingDir     string            `json:"workingDir,omitempty"`
	ExtraEnv       map[string]string `json:"extraEnv,omitempty"`
	ScheduleMode   string            `json:"scheduleMode"`
	Interval       string            `json:"interval,omitempty"`
	CronExpr       string            `json:"cronExpr,omitempty"`
	Enabled        bool              `json:"enabled"`
	Timeout        string            `json:"timeout,omitempty"`
	Status         string            `json:"status"`
	PID            int               `json:"pid,omitempty"`
	LastStartedAt  string            `json:"lastStartedAt,omitempty"`
	LastFinishedAt string            `json:"lastFinishedAt,omitempty"`
	LastExitCode   *int              `json:"lastExitCode,omitempty"`
	LastError      string            `json:"lastError,omitempty"`
	NextRunAt      string            `json:"nextRunAt,omitempty"`
	LogPath        string            `json:"logPath"`
	RecentRuns     []CronTaskRun     `json:"recentRuns,omitempty"`
	CreatedAt      string            `json:"createdAt,omitempty"`
	UpdatedAt      string            `json:"updatedAt,omitempty"`
}

// ListCronTasks returns all global cron tasks.
func (c *Client) ListCronTasks() ([]CronTaskStatus, error) {
	var out []CronTaskStatus
	if err := c.getJSON("/api/cron-tasks", &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []CronTaskStatus{}
	}
	return out, nil
}

// CreateCronTask POSTs a new definition.
func (c *Client) CreateCronTask(def CronTaskDefinition) (*CronTaskStatus, error) {
	return c.saveCronTask(http.MethodPost, def)
}

// UpdateCronTask PUTs an existing definition (id required).
func (c *Client) UpdateCronTask(def CronTaskDefinition) (*CronTaskStatus, error) {
	return c.saveCronTask(http.MethodPut, def)
}

func (c *Client) saveCronTask(method string, def CronTaskDefinition) (*CronTaskStatus, error) {
	body, err := json.Marshal(def)
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(method, "/api/cron-tasks", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, readAPIError(resp)
	}
	var out CronTaskStatus
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode cron-tasks response: %w", err)
	}
	return &out, nil
}

// DeleteCronTask deletes by id.
func (c *Client) DeleteCronTask(id string) error {
	req, err := c.NewRequest(http.MethodDelete, "/api/cron-tasks?id="+url.QueryEscape(id), nil)
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return readAPIError(resp)
	}
	return nil
}

// EnableCronTask enables a task by id.
func (c *Client) EnableCronTask(id string) (*CronTaskStatus, error) {
	return c.postCronAction("/api/cron-tasks/enable", id)
}

// DisableCronTask disables a task by id.
func (c *Client) DisableCronTask(id string) (*CronTaskStatus, error) {
	return c.postCronAction("/api/cron-tasks/disable", id)
}

// RunCronTask manually fires a task.
func (c *Client) RunCronTask(id string) (*CronTaskStatus, error) {
	return c.postCronAction("/api/cron-tasks/run", id)
}

func (c *Client) postCronAction(path, id string) (*CronTaskStatus, error) {
	req, err := c.NewRequest(http.MethodPost, path+"?id="+url.QueryEscape(id), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, readAPIError(resp)
	}
	var out CronTaskStatus
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		// enable/disable/run may return empty; tolerate
		return &CronTaskStatus{ID: id}, nil
	}
	return &out, nil
}

// CronTaskHistory returns last 7d runs for a task id.
func (c *Client) CronTaskHistory(id string) ([]CronTaskRun, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/cron-tasks/history?id="+url.QueryEscape(id), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, readAPIError(resp)
	}
	var out []CronTaskRun
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode history: %w", err)
	}
	if out == nil {
		out = []CronTaskRun{}
	}
	return out, nil
}

// FindCronTask resolves a name or id from the list.
func FindCronTask(tasks []CronTaskStatus, idOrName string) (*CronTaskStatus, error) {
	idOrName = strings.TrimSpace(idOrName)
	if idOrName == "" {
		return nil, fmt.Errorf("cron task target cannot be empty")
	}
	for i := range tasks {
		if tasks[i].ID == idOrName {
			t := tasks[i]
			return &t, nil
		}
	}
	var matches []CronTaskStatus
	for i := range tasks {
		if tasks[i].Name == idOrName {
			matches = append(matches, tasks[i])
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no cron task found for %q", idOrName)
	case 1:
		return &matches[0], nil
	default:
		ids := make([]string, 0, len(matches))
		for _, m := range matches {
			ids = append(ids, m.ID)
		}
		return nil, fmt.Errorf("cron task name %q is ambiguous; matching IDs: %s", idOrName, strings.Join(ids, ", "))
	}
}
