package client

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/xhd2015/ai-critic/server/bookmarks"
)

// BookmarksDocument is the versioned bookmarks tree from the server.
type BookmarksDocument = bookmarks.Document

// BookmarkNode is a folder or url node.
type BookmarkNode = bookmarks.Node

// GetBookmarks returns the full bookmarks tree.
func (c *Client) GetBookmarks() (*BookmarksDocument, error) {
	var out BookmarksDocument
	if err := c.getJSON("/api/bookmarks", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// AddBookmarkRequest is the body for POST /api/bookmarks.
type AddBookmarkRequest struct {
	ParentID string  `json:"parent_id,omitempty"`
	Type     string  `json:"type"`
	ID       string  `json:"id,omitempty"`
	Name     string  `json:"name"`
	URL      string  `json:"url,omitempty"`
	Browser  *string `json:"browser,omitempty"`
	Index    *int    `json:"index,omitempty"`
}

// AddBookmark creates a node under parent (default root).
func (c *Client) AddBookmark(req AddBookmarkRequest) (*BookmarkNode, error) {
	if strings.TrimSpace(req.ParentID) == "" {
		req.ParentID = "root"
	}
	var out BookmarkNode
	if err := c.sendJSON(http.MethodPost, "/api/bookmarks", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateBookmarkRequest is the body for PATCH /api/bookmarks?id=.
type UpdateBookmarkRequest struct {
	Name    *string `json:"name,omitempty"`
	URL     *string `json:"url,omitempty"`
	Browser *string `json:"browser,omitempty"`
}

// UpdateBookmark patches fields on an existing node.
func (c *Client) UpdateBookmark(id string, req UpdateBookmarkRequest) (*BookmarkNode, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	var out BookmarkNode
	if err := c.sendJSON(http.MethodPatch, "/api/bookmarks?id="+url.QueryEscape(id), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteBookmark removes a node by id.
func (c *Client) DeleteBookmark(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("id is required")
	}
	req, err := c.NewRequest(http.MethodDelete, "/api/bookmarks?id="+url.QueryEscape(id), nil)
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

// MoveBookmarkRequest is the body for POST /api/bookmarks/move.
type MoveBookmarkRequest struct {
	ID       string `json:"id"`
	ParentID string `json:"parent_id"`
	Index    *int   `json:"index,omitempty"`
}

// MoveBookmark reparents a node.
func (c *Client) MoveBookmark(req MoveBookmarkRequest) error {
	if strings.TrimSpace(req.ID) == "" {
		return fmt.Errorf("id is required")
	}
	if strings.TrimSpace(req.ParentID) == "" {
		req.ParentID = "root"
	}
	return c.sendJSON(http.MethodPost, "/api/bookmarks/move", req, nil)
}
