package client

import (
	"net/http"
	"net/url"
)

type ProjectGitStatus struct {
	IsClean     bool `json:"is_clean"`
	Uncommitted int  `json:"uncommitted"`
}

type ProjectInfo struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	RepoURL         string           `json:"repo_url"`
	Dir             string           `json:"dir"`
	SSHKeyID        string           `json:"ssh_key_id,omitempty"`
	UseSSH          bool             `json:"use_ssh"`
	GitUserConfigID string           `json:"git_user_config_id,omitempty"`
	GitUserName     string           `json:"git_user_name,omitempty"`
	GitUserEmail    string           `json:"git_user_email,omitempty"`
	CreatedAt       string           `json:"created_at"`
	DirExists       bool             `json:"dir_exists"`
	GitStatus       ProjectGitStatus `json:"git_status"`
	ParentID        string           `json:"parent_id,omitempty"`
	Readme          string           `json:"readme,omitempty"`
}

type ProjectUpdate struct {
	GitUserConfigID *string `json:"git_user_config_id,omitempty"`
	GitUserName     *string `json:"git_user_name,omitempty"`
	GitUserEmail    *string `json:"git_user_email,omitempty"`
}

func (c *Client) ListProjects() ([]ProjectInfo, error) {
	var out []ProjectInfo
	if err := c.getJSON("/api/projects?all=true", &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []ProjectInfo{}
	}
	return out, nil
}

func (c *Client) UpdateProject(projectID string, update ProjectUpdate) (*ProjectInfo, error) {
	var out ProjectInfo
	if err := c.sendJSON(http.MethodPatch, "/api/projects?id="+url.QueryEscape(projectID), update, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) SetProjectGitConfig(projectID string, identityID string, name string, email string) (*ProjectInfo, error) {
	return c.UpdateProject(projectID, ProjectUpdate{
		GitUserConfigID: stringPtr(identityID),
		GitUserName:     stringPtr(name),
		GitUserEmail:    stringPtr(email),
	})
}

func (c *Client) UnsetProjectGitConfig(projectID string) (*ProjectInfo, error) {
	return c.UpdateProject(projectID, ProjectUpdate{
		GitUserConfigID: stringPtr(""),
		GitUserName:     stringPtr(""),
		GitUserEmail:    stringPtr(""),
	})
}

func stringPtr(v string) *string {
	return &v
}
