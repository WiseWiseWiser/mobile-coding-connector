package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type CustomAgent struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	Mode            string            `json:"mode"`
	Model           string            `json:"model,omitempty"`
	Tools           map[string]bool   `json:"tools"`
	Permissions     map[string]string `json:"permissions,omitempty"`
	HasSystemPrompt bool              `json:"hasSystemPrompt"`
	SystemPrompt    string            `json:"systemPrompt,omitempty"`
}

type CreateCustomAgentRequest struct {
	ID           string            `json:"id,omitempty"`
	Name         string            `json:"name,omitempty"`
	Description  string            `json:"description,omitempty"`
	Mode         string            `json:"mode,omitempty"`
	Model        string            `json:"model,omitempty"`
	Tools        map[string]bool   `json:"tools,omitempty"`
	Permissions  map[string]string `json:"permissions,omitempty"`
	Template     string            `json:"template,omitempty"`
	SystemPrompt string            `json:"systemPrompt,omitempty"`
}

type UpdateCustomAgentRequest struct {
	Name         string            `json:"name,omitempty"`
	Description  string            `json:"description,omitempty"`
	Mode         string            `json:"mode,omitempty"`
	Model        string            `json:"model,omitempty"`
	Tools        map[string]bool   `json:"tools,omitempty"`
	Permissions  map[string]string `json:"permissions,omitempty"`
	SystemPrompt string            `json:"systemPrompt,omitempty"`
}

type CustomAgentSession struct {
	ID         string `json:"id"`
	AgentID    string `json:"agent_id"`
	AgentName  string `json:"agent_name"`
	ProjectDir string `json:"project_dir"`
	Port       int    `json:"port"`
	CreatedAt  string `json:"created_at"`
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
}

type LaunchCustomAgentResponse struct {
	SessionID string `json:"sessionId"`
	Port      int    `json:"port"`
	URL       string `json:"url"`
}

func (c *Client) ListCustomAgents() ([]CustomAgent, error) {
	var out []CustomAgent
	if err := c.getJSON("/api/custom-agents", &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []CustomAgent{}
	}
	return out, nil
}

func (c *Client) GetCustomAgent(agentID string) (*CustomAgent, error) {
	var out CustomAgent
	if err := c.getJSON("/api/custom-agents/"+url.PathEscape(agentID), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreateCustomAgent(reqBody CreateCustomAgentRequest) (*CustomAgent, error) {
	var out CustomAgent
	if err := c.postJSON("/api/custom-agents", reqBody, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateCustomAgent(agentID string, reqBody UpdateCustomAgentRequest) (*CustomAgent, error) {
	var out CustomAgent
	if err := c.sendJSON(http.MethodPut, "/api/custom-agents/"+url.PathEscape(agentID), reqBody, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteCustomAgent(agentID string) error {
	req, err := c.NewRequest(http.MethodDelete, "/api/custom-agents/"+url.PathEscape(agentID), nil)
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

func (c *Client) ListCustomAgentSessions(agentID string) ([]CustomAgentSession, error) {
	var out []CustomAgentSession
	if err := c.getJSON("/api/custom-agents/"+url.PathEscape(agentID)+"/sessions", &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []CustomAgentSession{}
	}
	return out, nil
}

func (c *Client) StopCustomAgentSession(sessionID string) error {
	req, err := c.NewRequest(http.MethodDelete, "/api/custom-agents/sessions/"+url.PathEscape(sessionID), nil)
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

func (c *Client) LaunchCustomAgent(agentID string, projectDir string, sessionID string) (*LaunchCustomAgentResponse, error) {
	reqBody := struct {
		ProjectDir string `json:"projectDir"`
		SessionID  string `json:"sessionId,omitempty"`
	}{
		ProjectDir: projectDir,
		SessionID:  sessionID,
	}

	var out LaunchCustomAgentResponse
	if err := c.postJSON("/api/custom-agents/"+url.PathEscape(agentID)+"/launch", reqBody, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) WaitCustomAgentSessionReady(sessionID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	path := "/api/custom-agents/sessions/" + url.PathEscape(sessionID) + "/proxy/session"

	for {
		req, err := c.NewRequest(http.MethodGet, path, nil)
		if err != nil {
			return err
		}
		resp, err := c.Do(req)
		if err == nil {
			resp.Body.Close()
			switch {
			case resp.StatusCode >= 200 && resp.StatusCode < 300:
				return nil
			case resp.StatusCode == http.StatusServiceUnavailable:
				// still starting
			case resp.StatusCode == http.StatusNotFound:
				return fmt.Errorf("session %s is not running", sessionID)
			default:
				return fmt.Errorf("wait for session ready: unexpected status %s", resp.Status)
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for custom agent session %s to become ready", sessionID)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func (c *Client) postJSON(path string, reqBody any, out any) error {
	return c.sendJSON(http.MethodPost, path, reqBody, out)
}

func (c *Client) sendJSON(method string, path string, reqBody any, out any) error {
	data, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal %s request: %w", path, err)
	}

	req, err := c.NewRequest(method, path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return readAPIError(resp)
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode %s response: %w", path, err)
	}
	return nil
}
