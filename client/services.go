package client

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type ServicePortForward struct {
	Port       int    `json:"port"`
	Label      string `json:"label,omitempty"`
	Provider   string `json:"provider,omitempty"`
	BaseDomain string `json:"baseDomain,omitempty"`
	Subdomain  string `json:"subdomain,omitempty"`
}

type ServicePortForwardStatus struct {
	Port       int    `json:"port"`
	Label      string `json:"label,omitempty"`
	Provider   string `json:"provider,omitempty"`
	BaseDomain string `json:"baseDomain,omitempty"`
	Subdomain  string `json:"subdomain,omitempty"`
	PublicURL  string `json:"publicUrl,omitempty"`
	Status     string `json:"status,omitempty"`
	Error      string `json:"error,omitempty"`
	Active     bool   `json:"active"`
}

type ServiceDefinition struct {
	ID                    string              `json:"id,omitempty"`
	Name                  string              `json:"name"`
	Command               string              `json:"command"`
	WorkingDir            string              `json:"workingDir,omitempty"`
	ExtraEnv              map[string]string   `json:"extraEnv,omitempty"`
	PortForward           *ServicePortForward `json:"portForward,omitempty"`
	UpgradeTarget         string              `json:"upgradeTarget,omitempty"`
	UpgradePreStopCmds    []string            `json:"upgradePreStopCmds,omitempty"`
	UpgradePostStopCmds   []string            `json:"upgradePostStopCmds,omitempty"`
	UpgradeTimeoutSeconds *int                `json:"upgradeTimeoutSeconds,omitempty"`
	Enabled               *bool               `json:"enabled,omitempty"`
	RequireAuth           bool                `json:"requireAuth,omitempty"`
	AuthUser              string              `json:"authUser,omitempty"`
	AuthTokenMode         string              `json:"authTokenMode,omitempty"`
	AuthToken             string              `json:"authToken,omitempty"`
}

type ServiceStatus struct {
	ID             string                    `json:"id"`
	Name           string                    `json:"name"`
	Kind           string                    `json:"kind,omitempty"`
	Description    string                    `json:"description,omitempty"`
	Command        string                    `json:"command"`
	WorkingDir     string                    `json:"workingDir,omitempty"`
	ExtraEnv       map[string]string         `json:"extraEnv,omitempty"`
	EffectivePath  string                    `json:"effectivePath,omitempty"`
	LogPath        string                    `json:"logPath"`
	Status         string                    `json:"status"`
	PID            int                       `json:"pid"`
	LastStartedAt  string                    `json:"lastStartedAt,omitempty"`
	LastExitedAt   string                    `json:"lastExitedAt,omitempty"`
	LastExitError  string                    `json:"lastExitError,omitempty"`
	DesiredRunning bool                      `json:"desiredRunning"`
	Enabled        bool                      `json:"enabled"`
	PortForward    *ServicePortForwardStatus `json:"portForward,omitempty"`
	UpgradeTarget  string                    `json:"upgradeTarget,omitempty"`
	// Upgrade step configuration, mirrored so `service update` round-trips it.
	UpgradePreStopCmds    []string `json:"upgradePreStopCmds,omitempty"`
	UpgradePostStopCmds   []string `json:"upgradePostStopCmds,omitempty"`
	UpgradeTimeoutSeconds *int     `json:"upgradeTimeoutSeconds,omitempty"`
	RequireAuth           bool     `json:"requireAuth,omitempty"`
	AuthUser              string   `json:"authUser,omitempty"`
	AuthTokenMode         string   `json:"authTokenMode,omitempty"`
	AuthToken             string   `json:"authToken,omitempty"`
	AuthTokens            []string `json:"authTokens,omitempty"`

	// System-service detail. Empty for user services.
	Detail    string `json:"detail,omitempty"`
	PublicURL string `json:"publicUrl,omitempty"`
	Port      int    `json:"port,omitempty"`
	Mocked    bool   `json:"mocked,omitempty"`
	// AutoStartSwitch reports that the subsystem exposes its own auto-start
	// flag, so the CLI and GUI offer enable/disable.
	AutoStartSwitch bool `json:"autoStartSwitch,omitempty"`
	// Edge is the upstream a proxying system service publishes through.
	Edge string `json:"edge,omitempty"`
	// Hosts lists the public hostnames a system service publishes.
	Hosts []SystemHostStatus `json:"hosts,omitempty"`
	// Actions lists the actions a system service supports. Empty for user
	// services, which always offer the full set.
	Actions []string `json:"actions,omitempty"`
}

// SystemHostStatus is one public hostname a system service publishes.
type SystemHostStatus struct {
	Host  string `json:"host"`
	Dials int    `json:"dials"`
	State string `json:"state"`
}

// ServicePreset is a predefined template for a user service.
type ServicePreset struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Command     string `json:"command"`
	Port        int    `json:"port,omitempty"`
}

type ServiceActionResponse struct {
	Status  string         `json:"status"`
	Message string         `json:"message"`
	Service *ServiceStatus `json:"service"`
}

type LogStreamEvent struct {
	Type    string `json:"type"`
	Message string `json:"message,omitempty"`
	Status  string `json:"status,omitempty"`
}

type ServiceUpgradeRequest struct {
	ID string `json:"id"`
	// TmpPath and LocalBase describe an uploaded binary; empty means a
	// step-only upgrade.
	TmpPath   string `json:"tmpPath,omitempty"`
	LocalBase string `json:"localBase,omitempty"`
	Target    string `json:"target,omitempty"`
	// One-run overrides of the stored steps.
	PreStopCmds    []string `json:"preStopCmds,omitempty"`
	PostStopCmds   []string `json:"postStopCmds,omitempty"`
	TimeoutSeconds *int     `json:"timeoutSeconds,omitempty"`
}

// ServiceUpgradeStep is one executed upgrade step.
type ServiceUpgradeStep struct {
	Phase      string `json:"phase"`
	Index      int    `json:"index"`
	Total      int    `json:"total"`
	Command    string `json:"command"`
	ExitCode   int    `json:"exitCode"`
	DurationMs int64  `json:"durationMs"`
}

type ServiceUpgradeResult struct {
	Status           string               `json:"status"`
	TmpPath          string               `json:"tmpPath,omitempty"`
	TargetPath       string               `json:"targetPath,omitempty"`
	RememberedTarget string               `json:"rememberedTarget,omitempty"`
	Service          *ServiceStatus       `json:"service,omitempty"`
	Steps            []ServiceUpgradeStep `json:"steps,omitempty"`
}

// ListServices returns every managed service.
func (c *Client) ListServices() ([]ServiceStatus, error) {
	var out []ServiceStatus
	if err := c.getJSON("/api/services", &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []ServiceStatus{}
	}
	return out, nil
}

// ListAllServices is an alias of ListServices (services are global).
func (c *Client) ListAllServices() ([]ServiceStatus, error) {
	return c.ListServices()
}

func (c *Client) StartService(id string) (*ServiceStatus, error) {
	req, err := c.NewRequest(http.MethodPost, "/api/services/start?id="+url.QueryEscape(id), nil)
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

	var out ServiceStatus
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode /api/services/start response: %w", err)
	}
	return &out, nil
}

func (c *Client) StopService(id string) error {
	return c.postServiceAction("/api/services/stop", id)
}

func (c *Client) RestartService(id string) error {
	return c.postServiceAction("/api/services/restart", id)
}

func (c *Client) DisableService(id string) (*ServiceActionResponse, error) {
	return c.postServiceActionWithResponse("/api/services/disable", id)
}

func (c *Client) EnableService(id string) (*ServiceActionResponse, error) {
	return c.postServiceActionWithResponse("/api/services/enable", id)
}

func (c *Client) SaveService(def ServiceDefinition, restart bool) (*ServiceStatus, error) {
	body, err := json.Marshal(def)
	if err != nil {
		return nil, err
	}
	path := "/api/services"
	if !restart {
		path += "?restart=false"
	}
	method := http.MethodPost
	if strings.TrimSpace(def.ID) != "" {
		method = http.MethodPut
	}
	req, err := c.NewRequest(method, path, bytes.NewReader(body))
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

	var out ServiceStatus
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode /api/services response: %w", err)
	}
	return &out, nil
}

// DeleteService removes a managed service by id.
// DELETE /api/services?id=<id>
func (c *Client) DeleteService(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("service id is required")
	}
	req, err := c.NewRequest(http.MethodDelete, "/api/services?id="+url.QueryEscape(id), nil)
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

func (c *Client) UpgradeService(upgrade ServiceUpgradeRequest) (*ServiceUpgradeResult, error) {
	body, err := json.Marshal(upgrade)
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(http.MethodPost, "/api/services/upgrade", bytes.NewReader(body))
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

	var out ServiceUpgradeResult
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode /api/services/upgrade response: %w", err)
	}
	return &out, nil
}

// UpgradeServiceStream runs an upgrade over SSE, reporting each phase and step
// output line through handler as it happens.
func (c *Client) UpgradeServiceStream(upgrade ServiceUpgradeRequest, handler func(ServerStreamEvent)) (*SSEStreamResult, error) {
	return c.StreamSSEWithDone("/api/services/upgrade/stream", upgrade, handler)
}

func (c *Client) postServiceAction(path string, id string) error {
	_, err := c.postServiceActionWithResponse(path, id)
	return err
}

func (c *Client) postServiceActionWithResponse(path string, id string) (*ServiceActionResponse, error) {
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

	var out ServiceActionResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode %s response: %w", path, err)
	}
	return &out, nil
}

func (c *Client) StreamLogFile(path string, lines int, handler func(LogStreamEvent)) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("log path is required")
	}
	if lines <= 0 {
		lines = 100
	}

	req, err := c.NewRequest(http.MethodGet, "/api/logs/stream?path="+url.QueryEscape(path)+"&lines="+url.QueryEscape(fmt.Sprintf("%d", lines)), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return readAPIError(resp)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		var ev LogStreamEvent
		if err := json.Unmarshal([]byte(payload), &ev); err != nil {
			return fmt.Errorf("decode log stream event: %w", err)
		}
		if handler != nil {
			handler(ev)
		}
		if ev.Type == "error" {
			return errors.New(defaultStreamError(ev.Message, "log stream failed"))
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read log stream: %w", err)
	}
	return nil
}
