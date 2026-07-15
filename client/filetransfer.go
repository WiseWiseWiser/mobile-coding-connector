package client

import (
	"bytes"
	"encoding/json"
	"net/http"
)

// ScratchEntry is the shared file-transfer scratch pad blob.
type ScratchEntry struct {
	Content   string `json:"content"`
	UpdatedAt string `json:"updated_at"`
}

// GetFileTransferScratch fetches the scratch pad via GET /api/file-transfer/scratch.
func (c *Client) GetFileTransferScratch() (*ScratchEntry, error) {
	var out ScratchEntry
	if err := c.getJSON("/api/file-transfer/scratch", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// PutFileTransferScratch overwrites the scratch pad via PUT /api/file-transfer/scratch.
func (c *Client) PutFileTransferScratch(content string) (*ScratchEntry, error) {
	body, err := json.Marshal(map[string]string{"content": content})
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(http.MethodPut, "/api/file-transfer/scratch", bytes.NewReader(body))
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

	var out ScratchEntry
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}