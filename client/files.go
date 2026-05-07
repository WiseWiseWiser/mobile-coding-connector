package client

import (
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
