# WS-Proxy Doctests

Package-level tests for `server/proxy/wsproxy` that verify client-ready status,
VMess link preconditions, and tunnel ingress consistency.

# DSN (Domain Specific Notion)

The ws-proxy doctest harness models the server-side Xray + Cloudflare tunnel
stack that exposes a VMess-over-WebSocket endpoint to mobile/desktop clients.

**Participants**

- **Manager** — in-memory `publicURL`, orchestrates xray and tunnel mapping.
- **xray inbound** — local VMess/WebSocket listener (health: HTTP GET `/ws` → 400).
- **Extension tunnel group** — Cloudflare ingress mapping `ws-proxy` hostname → localhost.
- **VMess clients** — V2RayU, v2rayNG, Shadowrocket import the `vmess://` link.

**Behaviors**

- `Status().Running` is true when the xray subprocess is tracked OR `isXrayAlive` succeeds.
- `GetVMessLink()` requires `publicURL`, `UUID`, and local xray.
- Permanent tunnels register `IngressMapping{ID: ws-proxy}` on the extension group.
- Clients fail with `ERR_PROXY_CONNECTION_FAILED` when the public hostname returns
  Cloudflare 404 (no ingress) even though local xray responds 400 on `/ws`.

## Decision Tree

```
[ws-proxy status correctness]
 |
 +-- status-without-tunnel-mapping/          (LEAF)
 |    xray alive + publicURL in memory, no ingress mapping
 |    → Status must NOT claim client-ready
 |
 +-- status-orphan-xray-no-public-url/       (LEAF)
 |    xray alive after restart, publicURL lost
 |    → publicURL must be reconstructed from persisted config
 |
 +-- vmess-link-requires-tunnel-ready/       (LEAF)
      tunnel mapping absent
      → vmess-link API must refuse / return not-ready
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `status-without-tunnel-mapping` | Running=true with local xray but missing Cloudflare ingress must not be client-ready |
| 2 | `status-orphan-xray-no-public-url` | After restart, publicURL must be derived from ws-proxy.json + domain config |
| 3 | `vmess-link-requires-tunnel-ready` | VMess link must not be served when tunnel ingress is absent |

## How to Run

```sh
doctest test ./server/proxy/wsproxy/tests/...
go test ./server/proxy/wsproxy/... -run 'Test(Status|VMess|Tunnel)'
```

```go
import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xhd2015/ai-critic/server/cloudflare/unified_tunnel"
	"github.com/xhd2015/ai-critic/server/proxy/wsproxy"
)

type Request struct {
	WSPath           string
	PublicURL        string
	InstanceID       string
	Subdomain        string
	UUID             string
	SimulateXray     bool
	AddTunnelMapping bool
	ClearPublicURL   bool
}

type Response struct {
	StatusRunning        bool
	StatusPublicURL      string
	TunnelMappingPresent bool
	VMessLink            string
	LocalXrayAlive       bool
	ClientReady          bool
}

func Run(t *testing.T, req *Request) (*Response, error) {
	if req.WSPath == "" {
		req.WSPath = "/ws"
	}
	if req.Subdomain == "" {
		req.Subdomain = "ws"
	}
	if req.UUID == "" {
		req.UUID = "00000000-0000-4000-8000-000000000001"
	}
	if req.InstanceID == "" {
		req.InstanceID = "25b2a55939e4"
	}
	if req.PublicURL == "" {
		req.PublicURL = "https://ws-25b2a55939e4.xhd2015.xyz"
	}

	tmpDir := t.TempDir()
	cfg := &wsproxy.Config{
		UpstreamProxy: "http://proxy.internal:3128",
		ListenPort:    0,
		WSPath:        req.WSPath,
		UUID:          req.UUID,
		Subdomain:     req.Subdomain,
		InstanceID:    req.InstanceID,
		AutoStart:     true,
	}
	if req.ClearPublicURL {
		cfg.PublicURL = "https://ws-25b2a55939e4.xhd2015.xyz"
	}

	var xraySrv *httptest.Server
	if req.SimulateXray {
		xraySrv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == req.WSPath {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			http.NotFound(w, r)
		}))
		defer xraySrv.Close()
		port := wsproxy.ExtractPortFromURL(xraySrv.URL)
		cfg.ListenPort = port
	}

	wsproxy.SetTestConfigDir(tmpDir)
	t.Cleanup(func() { wsproxy.SetTestConfigDir("") })
	if err := wsproxy.SaveTestConfig(cfg); err != nil {
		return nil, err
	}

	publicURL := req.PublicURL
	if req.ClearPublicURL {
		publicURL = ""
	}

	m := wsproxy.NewTestManager(publicURL, false)

	hostname := wsproxy.HostFromPublicURL(req.PublicURL)
	if req.AddTunnelMapping {
		tg := unified_tunnel.GetTunnelGroupManager().GetExtensionGroup()
		_ = tg.AddMapping(&unified_tunnel.IngressMapping{
			ID:       "ws-proxy",
			Hostname: hostname,
			Service:  "http://localhost:1",
			Source:   "wsproxy-test",
		})
		t.Cleanup(func() { _ = tg.RemoveMapping("ws-proxy") })
	}

	status := m.Status()
	resp := &Response{
		StatusRunning:   status.Running,
		StatusPublicURL: status.PublicURL,
		VMessLink:       m.GetVMessLink(),
		LocalXrayAlive:  wsproxy.IsXrayAliveForTest(cfg.ListenPort, req.WSPath),
	}

	tg := unified_tunnel.GetTunnelGroupManager().GetExtensionGroup()
	for _, mapping := range tg.ListMappings() {
		if mapping.ID == "ws-proxy" && mapping.Hostname == hostname {
			resp.TunnelMappingPresent = true
			break
		}
	}

	resp.ClientReady = resp.StatusRunning && resp.StatusPublicURL != "" &&
		resp.TunnelMappingPresent && resp.LocalXrayAlive && resp.VMessLink != ""

	return resp, nil
}
```