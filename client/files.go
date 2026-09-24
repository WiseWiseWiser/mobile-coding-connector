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
	// MD5 is the content digest. It is set only by CheckPathMD5, and stays
	// empty for missing paths, directories, and servers that predate digest
	// reporting.
	MD5 string `json:"md5,omitempty"`
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
	return c.checkPath(path, false)
}

// CheckPathMD5 is CheckPath plus the file's md5 digest, computed on the server.
// The digest is empty when the path is missing or is a directory, and when the
// server does not support digest reporting (older builds ignore the request
// field), which callers must treat as "digest unavailable" rather than "empty
// file".
func (c *Client) CheckPathMD5(path string) (*PathInfo, error) {
	return c.checkPath(path, true)
}

func (c *Client) checkPath(path string, wantMD5 bool) (*PathInfo, error) {
	payload := map[string]any{"path": path}
	if wantMD5 {
		payload["md5"] = true
	}
	body, err := json.Marshal(payload)
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