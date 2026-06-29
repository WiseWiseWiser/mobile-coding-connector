package client

import (
	"net/http"
	"net/url"
)

type ProjectGitStatus struct {
	IsClean       bool   `json:"is_clean"`
	Branch        string `json:"branch,omitempty"`
	Commit        string `json:"commit,omitempty"`
	CommitMessage string `json:"commit_message,omitempty"`
	Added         int    `json:"added"`
	Changed       int    `json:"changed"`
	Renamed       int    `json:"renamed"`
	Deleted       int    `json:"deleted"`
	Uncommitted   int    `json:"uncommitted"`
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

type ProjectListOptions struct {
	DirtyOnly bool
}

func (c *Client) ListProjects(opts ProjectListOptions) ([]ProjectInfo, error) {
	query := url.Values{}
	query.Set("all", "true")
	if opts.DirtyOnly {
		query.Set("dirty", "true")
	}
	var out []ProjectInfo
	if err := c.getJSON("/api/projects?"+query.Encode(), &out); err != nil {
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
