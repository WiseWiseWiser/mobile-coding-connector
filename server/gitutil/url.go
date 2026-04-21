// Package gitutil holds small helpers for manipulating git repository
// URLs that are shared across server subpackages.
package gitutil

import (
	"net/url"
	"strings"
)

// IsSSH reports whether repo is already an SSH-style git URL.
// Recognised forms:
//   - ssh://[user@]host[:port]/path
//   - user@host:path   (scp-like short form)
func IsSSH(repo string) bool {
	s := strings.TrimSpace(repo)
	if strings.HasPrefix(s, "ssh://") {
		return true
	}
	if strings.Contains(s, "://") {
		return false
	}
	// scp-like: must have a single ':' separating host and path, and
	// a '@' on the host side (i.e. before the colon). A plain local
	// path like "some:thing" without an '@' is not an SSH URL.
	colon := strings.Index(s, ":")
	if colon < 0 {
		return false
	}
	host := s[:colon]
	return strings.Contains(host, "@") && !strings.Contains(host, "/")
}

// ToSSH converts an HTTPS-style git URL to the scp-like SSH form used
// by GitLab/GitHub/Bitbucket, i.e.:
//
//	https://host/owner/name(.git)  ->  <sshUser>@host:owner/name(.git)
//
// If repo is already SSH (ssh:// or scp-like), it is returned
// unchanged. If repo can't be parsed as HTTPS/HTTP with a host, it is
// also returned unchanged — callers can still attempt to clone it and
// git will report the error.
//
// sshUser is the user component of the SSH URL. It defaults to "git"
// when empty. GitLab instances that reject the "git" user should pass
// "gitlab" (e.g. git.garena.com).
func ToSSH(repo string, sshUser string) string {
	s := strings.TrimSpace(repo)
	if s == "" {
		return repo
	}
	if IsSSH(s) {
		return s
	}
	if !strings.Contains(s, "://") {
		return repo
	}
	u, err := url.Parse(s)
	if err != nil {
		return repo
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return repo
	}
	host := u.Host
	if host == "" {
		return repo
	}
	// Drop any port: SSH uses port 22 regardless of what the HTTPS URL
	// advertised. Callers that need a non-standard SSH port must pass
	// an SSH URL directly.
	if i := strings.Index(host, ":"); i >= 0 {
		host = host[:i]
	}
	path := strings.TrimPrefix(u.Path, "/")
	if path == "" {
		return repo
	}
	user := strings.TrimSpace(sshUser)
	if user == "" {
		user = "git"
	}
	return user + "@" + host + ":" + path
}
