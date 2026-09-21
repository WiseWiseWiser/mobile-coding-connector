package gomodrelay

import (
	"net/url"
	"strings"
)

// upstreamURL joins the upstream base URL with the incoming request path/query.
func upstreamURL(upstream string, in *url.URL) string {
	base, err := url.Parse(upstream)
	if err != nil {
		return upstream + in.Path
	}
	base.Path = joinPath(base.Path, in.Path)
	base.RawQuery = in.RawQuery
	return base.String()
}

func joinPath(a, b string) string {
	as := strings.TrimSuffix(a, "/")
	bs := strings.TrimPrefix(b, "/")
	if as == "" {
		return "/" + bs
	}
	if bs == "" {
		return as
	}
	return as + "/" + bs
}

// upstreamHost extracts host:port from the upstream URL, defaulting to port 80.
func upstreamHost(upstream string) string {
	u, err := url.Parse(upstream)
	if err != nil || u.Host == "" {
		return upstream
	}
	if u.Port() != "" {
		return u.Host
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return u.Host + ":443"
	default:
		return u.Host + ":80"
	}
}
