package cloudflareproxy

import "net/http"

// WireRequest is one visitor HTTP request sent over a dial WebSocket.
type WireRequest struct {
	Method string              `json:"method"`
	URL    string              `json:"url"`
	Host   string              `json:"host"`
	Header map[string][]string `json:"header"`
	Body   []byte              `json:"body"`
}

// WireResponse is the origin/echo HTTP response sent back over the same WS.
type WireResponse struct {
	Status int                 `json:"status"`
	Header map[string][]string `json:"header"`
	Body   []byte              `json:"body"`
}

var hopHeaders = map[string]bool{
	"Connection":          true,
	"Keep-Alive":          true,
	"Proxy-Authenticate":  true,
	"Proxy-Authorization": true,
	"Te":                  true,
	"Trailers":            true,
	"Transfer-Encoding":   true,
	"Upgrade":             true,
}

func cloneHeader(h http.Header) map[string][]string {
	if h == nil {
		return nil
	}
	out := make(map[string][]string, len(h))
	for k, v := range h {
		if hopHeaders[http.CanonicalHeaderKey(k)] {
			continue
		}
		cp := make([]string, len(v))
		copy(cp, v)
		out[k] = cp
	}
	return out
}

func headerFromWire(h map[string][]string) http.Header {
	out := make(http.Header, len(h))
	for k, v := range h {
		if hopHeaders[http.CanonicalHeaderKey(k)] {
			continue
		}
		for _, x := range v {
			out.Add(k, x)
		}
	}
	return out
}
