// Package proxyselect picks a proxy from the server's saved
// proxyconfig for a given git repository.
//
// It is a thin layer above proxyconfig that (a) derives a hostname from
// a repo URL or a local repo's origin remote, and (b) formats a short
// credential-free note describing the selection so handlers can log it
// back to clients.
package proxyselect

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/xhd2015/lifelog-private/ai-critic/server/gitrunner"
	"github.com/xhd2015/lifelog-private/ai-critic/server/proxy/proxyconfig"
)

// Resolved carries both the proxy URL to apply (URL) and a short,
// credential-free description of how it was chosen (Note). Note is
// empty when no proxy should be applied and also when the caller
// supplied the URL explicitly (in that case there is nothing new to
// tell the user).
type Resolved struct {
	URL  string
	Note string
}

// ForRepoURL returns the proxy URL to apply to a git process targeting
// repo, plus a short log-worthy note describing the selection.
//
// Precedence:
//  1. An explicit URL wins; no note is emitted.
//  2. Otherwise, the host is parsed out of repo and matched against the
//     proxies saved in settings via proxyconfig.SelectProxyForHost.
//     When multiple proxies match the host, the last one is selected.
//     The note explains which proxy was picked.
//  3. If nothing matches (or the repo URL can't be parsed, or proxies
//     are disabled), URL is empty and no proxy is applied. The note is
//     empty in that case too.
func ForRepoURL(explicit string, repo string) Resolved {
	if explicit != "" {
		return Resolved{URL: explicit}
	}
	host := RepoHost(repo)
	if host == "" {
		return Resolved{}
	}
	cfg, err := proxyconfig.LoadConfig()
	if err != nil {
		return Resolved{}
	}
	p, ok := cfg.SelectProxyForHost(host)
	if !ok {
		return Resolved{}
	}
	label := p.Name
	if label == "" {
		label = p.ID
	}
	return Resolved{
		URL:  p.ProxyURL(),
		Note: fmt.Sprintf("auto-selected proxy %q (%s) for host %s", label, redactedProxyURL(p), host),
	}
}

// ForRepoDir is like ForRepoURL but discovers the repository URL by
// running 'git -C dir remote get-url origin'. A dir that isn't a git
// repo (or lacks an 'origin' remote) resolves to Resolved{}, which
// means no proxy is applied.
func ForRepoDir(explicit string, dir string) Resolved {
	if explicit != "" {
		return Resolved{URL: explicit}
	}
	return ForRepoURL("", OriginURL(dir))
}

// OriginURL returns the URL of the 'origin' remote in dir, or "" when
// that can't be determined.
func OriginURL(dir string) string {
	out, err := gitrunner.NewCommand("remote", "get-url", "origin").Dir(dir).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// RepoHost extracts the hostname from a git URL. It understands the
// three common forms:
//   - https://host/owner/name(.git)
//   - ssh://git@host/owner/name(.git)
//   - git@host:owner/name(.git)
//
// Returns "" when the URL has no discernible host (e.g. a bare local
// path).
func RepoHost(repo string) string {
	s := strings.TrimSpace(repo)
	if s == "" {
		return ""
	}
	if strings.Contains(s, "://") {
		u, err := url.Parse(s)
		if err != nil {
			return ""
		}
		return strings.ToLower(u.Hostname())
	}
	if idx := strings.Index(s, ":"); idx >= 0 && !strings.Contains(s[:idx], "/") {
		host := s[:idx]
		if at := strings.LastIndex(host, "@"); at >= 0 {
			host = host[at+1:]
		}
		return strings.ToLower(host)
	}
	return ""
}

// redactedProxyURL is like p.ProxyURL but strips credentials so the
// result is safe to log.
func redactedProxyURL(p *proxyconfig.ProxyServer) string {
	if p == nil {
		return ""
	}
	stripped := *p
	stripped.Username = ""
	stripped.Password = ""
	return stripped.ProxyURL()
}
