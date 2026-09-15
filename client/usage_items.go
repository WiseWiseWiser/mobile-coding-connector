package client

import (
	"fmt"
	"net/http"
	"strings"
)

// UsageItem is one registered menu-bar usage source.
type UsageItem struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Kind    string `json:"kind"`
	Enabled bool   `json:"enabled"`
	Home    string `json:"home,omitempty"`
	APIURL  string `json:"api_url,omitempty"`
}

// UsageItemView is a usage item plus the server-rendered menu text.
type UsageItemView struct {
	UsageItem
	Status    string `json:"status"`
	Title     string `json:"title"`
	Dropdown  string `json:"dropdown"`
	Detail    string `json:"detail,omitempty"`
	UsageURL  string `json:"usage_url,omitempty"`
	Error     string `json:"error,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// UsageItemsResponse is the GET /api/usage/items payload.
type UsageItemsResponse struct {
	Version int             `json:"version"`
	Default string          `json:"default"`
	Rotate  bool            `json:"rotate"`
	Items   []UsageItemView `json:"items"`
}

// UsageItemAddRequest is the body for POST /api/usage/items/add.
type UsageItemAddRequest struct {
	ID           string `json:"id,omitempty"`
	Label        string `json:"label,omitempty"`
	Kind         string `json:"kind,omitempty"`
	Home         string `json:"home,omitempty"`
	APIURL       string `json:"api_url,omitempty"`
	Enabled      *bool  `json:"enabled,omitempty"`
	Default      bool   `json:"default,omitempty"`
	SkipValidate bool   `json:"skip_validate,omitempty"`
	Strict       bool   `json:"strict,omitempty"`
}

// UsageItemUpdateRequest is the body for POST /api/usage/items/update.
type UsageItemUpdateRequest struct {
	ID           string  `json:"id"`
	Label        *string `json:"label,omitempty"`
	Kind         *string `json:"kind,omitempty"`
	Home         *string `json:"home,omitempty"`
	APIURL       *string `json:"api_url,omitempty"`
	Enabled      *bool   `json:"enabled,omitempty"`
	SkipValidate bool    `json:"skip_validate,omitempty"`
	Strict       bool    `json:"strict,omitempty"`
}

// UsageItemResult is the reply to add/update/remove.
type UsageItemResult struct {
	Item     *UsageItemView `json:"item,omitempty"`
	Warnings []string       `json:"warnings,omitempty"`
	Removed  string         `json:"removed,omitempty"`
	Default  string         `json:"default,omitempty"`
}

// ListUsageItems returns every usage item with rendered menu text.
func (c *Client) ListUsageItems() (*UsageItemsResponse, error) {
	var out UsageItemsResponse
	if err := c.getJSON("/api/usage/items", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// AddUsageItem registers a usage item.
func (c *Client) AddUsageItem(req UsageItemAddRequest) (*UsageItemResult, error) {
	var out UsageItemResult
	if err := c.sendJSON(http.MethodPost, "/api/usage/items/add", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateUsageItem changes an existing usage item.
func (c *Client) UpdateUsageItem(req UsageItemUpdateRequest) (*UsageItemResult, error) {
	if strings.TrimSpace(req.ID) == "" {
		return nil, fmt.Errorf("id is required")
	}
	var out UsageItemResult
	if err := c.sendJSON(http.MethodPost, "/api/usage/items/update", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// RemoveUsageItem deletes a usage item.
func (c *Client) RemoveUsageItem(id string) (*UsageItemResult, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	var out UsageItemResult
	if err := c.sendJSON(http.MethodPost, "/api/usage/items/remove", map[string]string{"id": id}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SetUsageDefault selects the menu-bar item, or rotation over enabled items.
func (c *Client) SetUsageDefault(id string, rotate bool) (*UsageItemsResponse, error) {
	var out UsageItemsResponse
	if err := c.sendJSON(http.MethodPost, "/api/usage/items/default", map[string]any{
		"id":     strings.TrimSpace(id),
		"rotate": rotate,
	}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
