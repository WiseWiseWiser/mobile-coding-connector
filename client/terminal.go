package client

import (
	"fmt"
	"net/http"
	"net/url"
)

type TerminalSession struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Cwd       string `json:"cwd"`
	CreatedAt string `json:"created_at"`
	Status    string `json:"status"`
	Connected bool   `json:"connected"`
}

type TerminalSessionsPage struct {
	Sessions   []TerminalSession `json:"sessions"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
	Total      int               `json:"total"`
	TotalPages int               `json:"total_pages"`
}

func (c *Client) ListTerminalSessions() ([]TerminalSession, error) {
	page := 1
	var sessions []TerminalSession

	for {
		var out TerminalSessionsPage
		path := fmt.Sprintf("/api/terminal/sessions?page=%d&page_size=100", page)
		if err := c.getJSON(path, &out); err != nil {
			return nil, err
		}
		sessions = append(sessions, out.Sessions...)
		if out.TotalPages <= page || len(out.Sessions) == 0 {
			break
		}
		page++
	}
	if sessions == nil {
		sessions = []TerminalSession{}
	}
	return sessions, nil
}

func (c *Client) DeleteTerminalSession(id string) error {
	req, err := c.NewRequest(http.MethodDelete, "/api/terminal/sessions?id="+url.QueryEscape(id), nil)
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
