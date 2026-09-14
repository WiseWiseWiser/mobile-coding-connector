package client

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// ResolveGrokSessionWorkspace finds the workspace directory for a Grok session id
// by scanning ~/.grok/sessions/<urlencoded-cwd>/<session-id>/ on the server.
// Returns ("", nil) when the session is not found (caller may fall back to home).
func (c *Client) ResolveGrokSessionWorkspace(sessionID string) (string, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return "", fmt.Errorf("session id required")
	}
	home, err := c.GetHome()
	if err != nil {
		return "", err
	}
	root := filepath.ToSlash(filepath.Join(home.Home, ".grok", "sessions"))
	listing, err := c.BrowseDir(root)
	if err != nil {
		return "", fmt.Errorf("browse %s: %w", root, err)
	}
	for _, ent := range listing.Entries {
		if !ent.IsDir {
			continue
		}
		candidate := filepath.ToSlash(filepath.Join(root, ent.Name, sessionID))
		info, err := c.CheckPath(candidate)
		if err != nil {
			return "", err
		}
		if info == nil || !info.Exists || !info.IsDir {
			continue
		}
		decoded, err := url.PathUnescape(ent.Name)
		if err != nil || strings.TrimSpace(decoded) == "" {
			return "", fmt.Errorf("decode workspace from %q: %w", ent.Name, err)
		}
		return filepath.Clean(decoded), nil
	}
	return "", nil
}
