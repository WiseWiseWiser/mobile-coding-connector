package client

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type GitUserConfig struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	CreatedAt string `json:"createdAt"`
}

type gitUserConfigRequest struct {
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Email     string          `json:"email,omitempty"`
	CreatedAt string          `json:"createdAt,omitempty"`
	Configs   []GitUserConfig `json:"configs,omitempty"`
}

func (c *Client) ListGitUserConfigs() ([]GitUserConfig, error) {
	var out []GitUserConfig
	if err := c.getJSON("/api/settings/git-user-configs", &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []GitUserConfig{}
	}
	return out, nil
}

func (c *Client) SaveGitUserConfigs(configs []GitUserConfig) ([]GitUserConfig, error) {
	var out []GitUserConfig
	if err := c.sendJSON(http.MethodPut, "/api/settings/git-user-configs", gitUserConfigRequest{Configs: configs}, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []GitUserConfig{}
	}
	return out, nil
}

func (c *Client) AddGitUserConfig(id string, name string, email string) (*GitUserConfig, error) {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if email == "" {
		return nil, fmt.Errorf("email is required")
	}

	var out GitUserConfig
	if err := c.sendJSON(http.MethodPost, "/api/settings/git-user-configs", gitUserConfigRequest{
		ID:    strings.TrimSpace(id),
		Name:  name,
		Email: email,
	}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateGitUserConfig(id string, name string, email string) (*GitUserConfig, error) {
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if email == "" {
		return nil, fmt.Errorf("email is required")
	}

	var out GitUserConfig
	if err := c.sendJSON(http.MethodPatch, "/api/settings/git-user-configs?id="+url.QueryEscape(id), gitUserConfigRequest{
		Name:  name,
		Email: email,
	}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteGitUserConfig(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("id is required")
	}
	req, err := c.NewRequest(http.MethodDelete, "/api/settings/git-user-configs?id="+url.QueryEscape(id), nil)
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
