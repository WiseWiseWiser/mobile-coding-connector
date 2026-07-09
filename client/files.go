package client

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
)

// BrowseEntry is one entry returned by /api/files/browse.
type BrowseEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size"`
}

// BrowseResult is the response returned by /api/files/browse.
type BrowseResult struct {
	Path    string        `json:"path"`
	Entries []BrowseEntry `json:"entries"`
}

// PathInfo is the response returned by /api/files/check.
type PathInfo struct {
	Exists bool   `json:"exists"`
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	IsDir  bool   `json:"is_dir"`
}

func (c *Client) BrowseDir(path string) (*BrowseResult, error) {
	var out BrowseResult
	if err := c.getJSON("/api/files/browse?path="+url.QueryEscape(path), &out); err != nil {
		return nil, err
	}
	if out.Entries == nil {
		out.Entries = []BrowseEntry{}
	}
	return &out, nil
}

// CheckPath reports whether path exists on the server and whether it is a directory.
func (c *Client) CheckPath(path string) (*PathInfo, error) {
	body, err := json.Marshal(map[string]string{"path": path})
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(http.MethodPost, "/api/files/check", bytes.NewReader(body))
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

	var out PathInfo
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}