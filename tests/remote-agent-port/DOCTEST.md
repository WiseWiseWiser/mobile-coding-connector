# Remote-Agent Port List + Ad-Hoc Visit Doctests

Doctests for `remote-agent port` (`list` / `visit`) and the server-side ad-hoc
visit session manager (`POST/GET/DELETE /api/ports/visit`).

All leaves are **L2 in-process**: `agentcli.Run` and/or
`portforward.VisitSessionManager` with **fake tunnel Providers** and an
injectable clock. No live cloudflared, no product binary e2e.

# DSN (Domain Specific Notion)

**L2 only** (unlabeled mass). Fake Providers implement
`portforward.Provider`; the visit manager starts an **ephemeral reverse-proxy**
hop, then tunnels the **proxy port** (not the raw app port). Idle shutdown is
in-memory (no durable visit JSON).

**Participants**

- **CLI (`agentcli.Run`)** — `remote-agent port list|visit…` against an ephemeral
  local mux with bearer auth (`lib.TestPassword`).
- **VisitSessionManager** — in-memory ad-hoc sessions; registers
  `POST/GET/DELETE /api/ports/visit` and owns reverse-proxy listeners.
- **Fake Providers** — `cloudflare_owned` and/or `cloudflare_quick` with
  controllable `Available()` and captured `Start(port, hostname)`.
- **Fake clock** — `SetNow` / `Sweep` so idle expiry is deterministic without
  sleeping 10 minutes.
- **Local ports seed** — L2 handler for `GET /api/ports/local` returns leaf-seeded
  `LocalPortInfo` rows (no real `lsof`).
- **Persistent forwards seed** — optional in-memory `portforward.Manager` tunnels
  for `port list --forwards`.
- **Mapping-names path** — isolated temp file so owned ad-hoc visits must not
  write `port-mapping-names.json`.

**Behaviors**

- `port list` prints listening ports as PORT / PID / COMMAND (or `--json` array);
  empty → clear empty message / `[]`.
- `port list --forwards` also includes active **persistent** forwards.
- `port visit <port>` starts an ad-hoc public mapping: provider **auto** prefers
  owned when `Available()`, else quick; explicit `--provider owned|quick`.
- Visit is a **proxy hop**: tunnel `Start` receives the ephemeral **proxy** port.
- Default idle **10m** from **tunnel-ready** (zero traffic still expires);
  `--idle` overrides; HTTP through the proxy resets last-activity.
- Same local port while an ad-hoc session is active → **error**.
- Port not listening → **warning** on stderr, still starts (exit 0).
- Owned ad-hoc uses an ephemeral random subdomain and **does not** write
  port-mapping-names.
- Foreground (default) prints URL + provider and blocks until stop/idle;
  `--detach --json` prints JSON and exits 0 while the session remains.
- Help at `port`, `port list`, `port visit`; unknown subcommand / bad flag → Error.

## Version

0.0.2

## Decision Tree

```
[remote-agent port]
 |
 +-- help/                              (GROUP) help surfaces
 |    +-- port-root/                    (LEAF)  port --help
 |    +-- list/                         (LEAF)  port list --help
 |    +-- visit/                        (LEAF)  port visit --help
 |
 +-- list/                              (GROUP) port list
 |    +-- empty/                        (LEAF)  no listeners → empty message
 |    +-- with-listeners/               (LEAF)  table PORT/PID/COMMAND
 |    +-- json/                         (LEAF)  --json array of listeners
 |    +-- with-forwards/                (LEAF)  --forwards includes persistent
 |
 +-- visit/                             (GROUP) ad-hoc visit
 |    +-- provider/                     (GROUP) provider selection (manager)
 |    |    +-- auto-owned/              (LEAF)  auto + owned Available → owned
 |    |    +-- auto-quick-fallback/     (LEAF)  auto + owned down → quick
 |    |    +-- auto-neither-error/      (LEAF)  auto + neither → error
 |    |    +-- owned-unavailable/       (LEAF)  --provider owned unavailable
 |    |    +-- explicit-quick/          (LEAF)  --provider quick wins over owned
 |    |
 |    +-- idle/                         (GROUP) idle proxy + hop (manager)
 |    |    +-- reverse-proxy-hop/       (LEAF)  tunnel Start gets proxy port
 |    |    +-- traffic-resets-activity/ (LEAF)  HTTP resets last-activity
 |    |    +-- zero-traffic-expires/    (LEAF)  ready + idle → stop
 |    |    +-- traffic-extends/         (LEAF)  traffic near deadline extends
 |    |    +-- explicit-stop/           (LEAF)  Stop removes session + tunnel
 |    |
 |    +-- session/                      (GROUP) session rules (manager + CLI warn)
 |    |    +-- duplicate-port-error/    (LEAF)  second visit same port → error
 |    |    +-- list-active/             (LEAF)  List shows active sessions
 |    |    +-- stop-by-id/              (LEAF)  Stop by session id
 |    |    +-- stop-by-port/            (LEAF)  Stop by local port
 |    |    +-- not-listening-warn/      (LEAF)  CLI warn + still starts
 |    |
 |    +-- cli/                          (GROUP) CLI presentation
 |    |    +-- foreground-url/          (LEAF)  prints URL + provider (short idle)
 |    |    +-- detach-json/             (LEAF)  --detach --json exits 0; session stays
 |    |    +-- visit-list/              (LEAF)  port visit list empty message
 |    |    +-- invalid-port/            (LEAF)  port 0 → Error
 |    |    +-- invalid-port-high/       (LEAF)  port 99999 → Error
 |    |
 |    +-- mapping-names/                (GROUP) owned side effects
 |         +-- owned-no-write/          (LEAF)  owned visit does not write names file
 |
 +-- rejected/                          (GROUP) CLI surface errors
      +-- unknown-subcommand/           (LEAF)  port foo → Error
      +-- bad-flag/                     (LEAF)  port list --not-a-flag → Error
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `help/port-root` | `port --help` usage lists `list` and `visit` |
| 2 | `help/list` | `port list --help` documents flags |
| 3 | `help/visit` | `port visit --help` documents provider/idle/detach |
| 4 | `list/empty` | No listeners → empty message; exit 0 |
| 5 | `list/with-listeners` | Seeded ports → PORT/PID/COMMAND table |
| 6 | `list/json` | `--json` → JSON array of listeners |
| 7 | `list/with-forwards` | `--forwards` includes persistent forward row |
| 8 | `visit/provider/auto-owned` | auto + owned Available → `cloudflare_owned` |
| 9 | `visit/provider/auto-quick-fallback` | auto + owned down → `cloudflare_quick` |
| 10 | `visit/provider/auto-neither-error` | auto + neither Available → error |
| 11 | `visit/provider/owned-unavailable` | explicit owned unavailable → error |
| 12 | `visit/provider/explicit-quick` | explicit quick when owned also up → quick |
| 13 | `visit/idle/reverse-proxy-hop` | Provider.Start port == ProxyPort ≠ LocalPort |
| 14 | `visit/idle/traffic-resets-activity` | HTTP to proxy advances LastActivity |
| 15 | `visit/idle/zero-traffic-expires` | advance past idle from ready → session gone |
| 16 | `visit/idle/traffic-extends` | traffic then advance within new idle → stays |
| 17 | `visit/idle/explicit-stop` | Stop cleans session + stops tunnel handle |
| 18 | `visit/session/duplicate-port-error` | second Start same port → error |
| 19 | `visit/session/list-active` | List returns active session fields |
| 20 | `visit/session/stop-by-id` | Stop(id) removes session |
| 21 | `visit/session/stop-by-port` | Stop(port) removes session |
| 22 | `visit/session/not-listening-warn` | CLI visit non-listening → warning:, exit 0 |
| 23 | `visit/cli/foreground-url` | short idle foreground prints URL + provider |
| 24 | `visit/cli/detach-json` | `--detach --json` JSON + exit 0; session remains |
| 25 | `visit/cli/visit-list` | `port visit list` empty → clear empty message |
| 26 | `visit/cli/invalid-port` | port 0 → Error, non-zero |
| 27 | `visit/cli/invalid-port-high` | port 99999 → Error, non-zero |
| 28 | `visit/mapping-names/owned-no-write` | owned Start leaves mapping-names unchanged |
| 29 | `rejected/unknown-subcommand` | `port foo` → Error |
| 30 | `rejected/bad-flag` | unknown flag → Error |

## Parameter Coverage

| Factor | Leaves |
|--------|--------|
| Subcommand (help / list / visit / rejected) | help/*, list/*, visit/*, rejected/* |
| List empty vs seeded vs JSON vs forwards | list/* |
| Provider auto / owned / quick availability | visit/provider/* |
| Idle clock + traffic | visit/idle/* |
| Proxy hop (proxy vs app port) | visit/idle/reverse-proxy-hop |
| Session uniqueness / list / stop | visit/session/* |
| Not-listening warn | visit/session/not-listening-warn |
| Foreground vs detach + JSON | visit/cli/* |
| Invalid port | visit/cli/invalid-port |
| Mapping-names write policy | visit/mapping-names/owned-no-write |
| CLI parse errors | rejected/* |

## Locked product API (implementer)

Classic TDD: these symbols are **not** in product yet. First make them
compile (skeleton), then satisfy Asserts.

Package `github.com/xhd2015/ai-critic/server/proxy/portforward`:

| Symbol | Role |
|--------|------|
| `AdhocVisitSession` | `{ID, LocalPort, ProxyPort, PublicURL, Provider, Hostname, CreatedAt, LastActivity, IdleTimeout, Status}` |
| `NewVisitSessionManager()` | in-memory manager |
| `(*VisitSessionManager).RegisterProvider(Provider)` | inject fakes |
| `(*VisitSessionManager).SetNow(func() time.Time)` | fake clock |
| `(*VisitSessionManager).SetListeningChecker(func(int) bool)` | inject listen state |
| `(*VisitSessionManager).SetMappingNamesPath(string)` | isolate names file |
| `(*VisitSessionManager).Start(port int, provider string, idle time.Duration) (*AdhocVisitSession, error)` | `provider`: `""`/`"auto"` / `"owned"` / `"quick"` / full names |
| `(*VisitSessionManager).List() []AdhocVisitSession` | active sessions |
| `(*VisitSessionManager).Stop(idOrPort string) error` | by id or decimal port |
| `(*VisitSessionManager).Sweep()` | expire idle using `Now` |
| `(*VisitSessionManager).RegisterAPI(mux *http.ServeMux)` | `POST/GET/DELETE /api/ports/visit` |
| Default idle | `10 * time.Minute` when `idle == 0` |

CLI (`agentcli`): `port` with `list` + `visit` (+ `visit list` / `visit stop`).

HTTP body: `{ "port", "provider"?, "idle_seconds"? }` — do **not** overload `POST /api/ports`.

## How to Run

```sh
doctest vet ./tests/remote-agent-port
doctest test ./tests/remote-agent-port/...
```

Classic TDD: tree is **RED** until VisitSessionManager + `remote-agent port` exist.

```go
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/cmd/agentcli"
	"github.com/xhd2015/ai-critic/script/lib"
	"github.com/xhd2015/ai-critic/server/proxy/portforward"
	"github.com/xhd2015/doctest/session"
)

// agentcliInProcessMu serializes in-process agentcli.Run (stdout/stderr swaps).
var agentcliInProcessMu sync.Mutex

// LocalPortSeed is one GET /api/ports/local row.
type LocalPortSeed struct {
	Port    int
	PID     int
	PPID    int
	Command string
	Cmdline string
}

// ForwardSeed is one persistent forward shown with --forwards.
type ForwardSeed struct {
	LocalPort int
	Label     string
	PublicURL string
	Status    string
	Provider  string
}

// Request configures one leaf. Op selects the Run branch.
//
// Op values:
//   cli              — agentcli.Run(Args) against L2 mux
//   visit-start      — manager.Start
//   visit-list       — manager.List (optionally after Start)
//   visit-stop       — Start then Stop(StopTarget)
//   visit-duplicate  — Start twice on same port
//   visit-idle       — Start, optional traffic, Advance, Sweep
//   visit-proxy-hop  — Start; capture Provider.Start port
//   visit-mapping    — Start owned; read mapping-names file
type Request struct {
	Op string

	// CLI
	Args []string

	// Visit parameters
	Port        int
	Provider    string // auto|owned|quick|cloudflare_owned|cloudflare_quick|""
	Idle        time.Duration
	StopTarget  string // id or port string for visit-stop / CLI

	// Fake provider availability (manager leaves)
	OwnedAvailable bool
	QuickAvailable bool

	// Seeds for L2 HTTP / manager
	LocalPorts []LocalPortSeed
	Forwards   []ForwardSeed

	// Listening checker: if set, overrides LocalPorts-based check for Start.
	// nil → treat Port as listening when present in LocalPorts (or true if LocalPorts empty and not NotListening).
	NotListening bool

	// Idle simulation (visit-idle)
	AdvanceAfterReady time.Duration
	SendTraffic       bool // HTTP GET to reverse-proxy before Advance
	AdvanceAfterTraffic time.Duration
	CaptureActivityBefore bool // snapshot LastActivity before traffic

	// Provider Start capture (visit-proxy-hop / provider leaves)
	CaptureStartPort bool

	// Mapping names isolation
	SeedMappingNames map[string]string
}

type VisitInfo struct {
	ID           string
	LocalPort    int
	ProxyPort    int
	PublicURL    string
	Provider     string
	Hostname     string
	IdleSeconds  int
	Status       string
	LastActivity time.Time
	CreatedAt    time.Time
}

type Response struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Combined string

	// Manager
	Session    *VisitInfo
	Sessions   []VisitInfo
	StartErr   string
	StopErr    string
	ListAfter  []VisitInfo

	// Provider capture
	TunnelStartPort     int
	TunnelStartHostname string
	OwnedStopCount      int
	QuickStopCount      int

	// Idle / hop
	ActivityBefore time.Time
	ActivityAfter  time.Time
	ProxyHTTPStatus int
	SessionAliveAfterSweep bool

	// Mapping names
	MappingNamesPath  string
	MappingNamesAfter map[string]string

	// Server
	ServerURL  string
	ConfigHome string
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	resp := &Response{}
	if req.Op == "" {
		req.Op = "cli"
	}

	configHome, err := os.MkdirTemp("", "remote-agent-port-*")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(configHome) })
	resp.ConfigHome = configHome

	mappingPath := filepath.Join(configHome, "port-mapping-names.json")
	resp.MappingNamesPath = mappingPath
	if req.SeedMappingNames != nil {
		if err := writeMappingNames(mappingPath, req.SeedMappingNames); err != nil {
			return nil, err
		}
	} else {
		// Ensure parent exists; file may be absent (empty).
		_ = os.MkdirAll(configHome, 0o755)
	}

	switch req.Op {
	case "cli":
		return runCLI(t, req, resp, configHome, mappingPath)
	case "visit-start", "visit-list", "visit-stop", "visit-duplicate",
		"visit-idle", "visit-proxy-hop", "visit-mapping":
		return runManager(t, req, resp, mappingPath)
	default:
		return nil, fmt.Errorf("unknown Op %q", req.Op)
	}
}

func runCLI(t *testing.T, req *Request, resp *Response, configHome, mappingPath string) (*Response, error) {
	t.Helper()

	vm, owned, quick, err := newVisitManagerForTest(req, mappingPath)
	if err != nil {
		return nil, err
	}
	_ = owned
	_ = quick

	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/api/ports/local", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(localPortInfos(req.LocalPorts))
	})
	// Persistent forwards for --forwards (existing /api/ports shape).
	pfm := portforward.NewManager()
	for _, f := range req.Forwards {
		// Seed via fake provider + Add is heavy; serve a static list handler instead.
		_ = f
	}
	mux.HandleFunc("/api/ports", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		out := make([]map[string]interface{}, 0, len(req.Forwards))
		for _, f := range req.Forwards {
			status := f.Status
			if status == "" {
				status = portforward.StatusActive
			}
			out = append(out, map[string]interface{}{
				"localPort": f.LocalPort,
				"label":     f.Label,
				"publicUrl": f.PublicURL,
				"status":    status,
				"provider":  f.Provider,
				"type":      "port_forward",
			})
		}
		_ = json.NewEncoder(w).Encode(out)
	})
	_ = pfm
	vm.RegisterAPI(mux)

	handler := withBearerAuth(lib.TestPassword, mux)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}
	srv := &http.Server{Handler: handler}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	port := ln.Addr().(*net.TCPAddr).Port
	serverURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	resp.ServerURL = serverURL

	if err := waitHTTPReady(serverURL+"/ping", 10*time.Second); err != nil {
		return nil, err
	}

	args := req.Args
	if len(args) == 0 {
		args = []string{"port"}
	}
	argv := append([]string{"--server", serverURL, "--token", lib.TestPassword}, args...)

	exitCode, stdout, stderr, runErr := runAgentInProcess(argv)
	if runErr != nil {
		return nil, runErr
	}
	resp.ExitCode = exitCode
	resp.Stdout = stdout
	resp.Stderr = stderr
	resp.Combined = strings.TrimSpace(stdout + "\n" + stderr)

	// Post-CLI: capture manager sessions + mapping names for detach/list asserts.
	resp.Sessions = toVisitInfos(vm.List())
	resp.MappingNamesAfter = readMappingNames(mappingPath)
	if owned != nil {
		resp.OwnedStopCount = owned.StopCount()
		resp.TunnelStartPort = owned.LastStartPort()
		resp.TunnelStartHostname = owned.LastHostname()
	}
	if quick != nil {
		resp.QuickStopCount = quick.StopCount()
		if resp.TunnelStartPort == 0 {
			resp.TunnelStartPort = quick.LastStartPort()
			resp.TunnelStartHostname = quick.LastHostname()
		}
	}
	return resp, nil
}

func runManager(t *testing.T, req *Request, resp *Response, mappingPath string) (*Response, error) {
	t.Helper()

	vm, owned, quick, err := newVisitManagerForTest(req, mappingPath)
	if err != nil {
		return nil, err
	}

	idle := req.Idle
	// Leaves pass short idle; 0 means product default (10m) — tests that need
	// default leave Idle=0 and Advance past 10m via AdvanceAfterReady.

	port := req.Port
	if port == 0 && req.Op != "visit-list" {
		port = 18080
	}

	startOnce := func() (*portforward.AdhocVisitSession, error) {
		return vm.Start(port, req.Provider, idle)
	}

	switch req.Op {
	case "visit-start", "visit-proxy-hop", "visit-mapping":
		sess, err := startOnce()
		if err != nil {
			resp.StartErr = err.Error()
			captureProvider(resp, owned, quick)
			resp.MappingNamesAfter = readMappingNames(mappingPath)
			return resp, nil
		}
		resp.Session = toVisitInfo(sess)
		captureProvider(resp, owned, quick)
		resp.MappingNamesAfter = readMappingNames(mappingPath)
		resp.Sessions = toVisitInfos(vm.List())
		return resp, nil

	case "visit-list":
		if port > 0 || req.Provider != "" || req.Idle > 0 || !req.NotListening {
			// Start a session first when Port is set (list-active leaf).
			if req.Port > 0 {
				sess, err := startOnce()
				if err != nil {
					resp.StartErr = err.Error()
					return resp, nil
				}
				resp.Session = toVisitInfo(sess)
			}
		}
		resp.Sessions = toVisitInfos(vm.List())
		captureProvider(resp, owned, quick)
		return resp, nil

	case "visit-stop":
		sess, err := startOnce()
		if err != nil {
			resp.StartErr = err.Error()
			return resp, nil
		}
		resp.Session = toVisitInfo(sess)
		target := req.StopTarget
		if target == "" {
			target = sess.ID
		}
		if err := vm.Stop(target); err != nil {
			resp.StopErr = err.Error()
		}
		resp.ListAfter = toVisitInfos(vm.List())
		captureProvider(resp, owned, quick)
		return resp, nil

	case "visit-duplicate":
		sess, err := startOnce()
		if err != nil {
			resp.StartErr = err.Error()
			return resp, nil
		}
		resp.Session = toVisitInfo(sess)
		_, err2 := startOnce()
		if err2 != nil {
			resp.StartErr = err2.Error()
		} else {
			resp.StartErr = ""
		}
		resp.Sessions = toVisitInfos(vm.List())
		return resp, nil

	case "visit-idle":
		// Deterministic clock: start at t0, ready immediately (fake provider).
		t0 := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
		now := t0
		vm.SetNow(func() time.Time { return now })

		sess, err := startOnce()
		if err != nil {
			resp.StartErr = err.Error()
			return resp, nil
		}
		resp.Session = toVisitInfo(sess)
		captureProvider(resp, owned, quick)

		if req.CaptureActivityBefore || req.SendTraffic {
			resp.ActivityBefore = sess.LastActivity
		}

		if req.SendTraffic {
			// Move clock so LastActivity can strictly advance on the request.
			now = now.Add(time.Second)
			vm.SetNow(func() time.Time { return now })
			// Hit reverse proxy to reset activity.
			url := fmt.Sprintf("http://127.0.0.1:%d/", sess.ProxyPort)
			status, err := httpGetStatus(url)
			if err != nil {
				return nil, fmt.Errorf("proxy traffic: %w", err)
			}
			resp.ProxyHTTPStatus = status
			// Re-list to read updated LastActivity.
			for _, s := range vm.List() {
				if s.ID == sess.ID {
					resp.ActivityAfter = s.LastActivity
					break
				}
			}
			if req.AdvanceAfterTraffic > 0 {
				now = now.Add(req.AdvanceAfterTraffic)
				vm.SetNow(func() time.Time { return now })
			}
		}

		if req.AdvanceAfterReady > 0 {
			now = now.Add(req.AdvanceAfterReady)
			vm.SetNow(func() time.Time { return now })
		}
		vm.Sweep()
		after := vm.List()
		resp.ListAfter = toVisitInfos(after)
		alive := false
		for _, s := range after {
			if s.ID == sess.ID {
				alive = true
				break
			}
		}
		resp.SessionAliveAfterSweep = alive
		captureProvider(resp, owned, quick)
		return resp, nil
	}

	return nil, fmt.Errorf("unhandled manager Op %q", req.Op)
}

func newVisitManagerForTest(req *Request, mappingPath string) (*portforward.VisitSessionManager, *fakeProvider, *fakeProvider, error) {
	vm := portforward.NewVisitSessionManager()
	vm.SetMappingNamesPath(mappingPath)

	// Listening checker: NotListening forces false; else true if LocalPorts
	// empty or Port is in LocalPorts.
	vm.SetListeningChecker(func(p int) bool {
		if req.NotListening {
			return false
		}
		if len(req.LocalPorts) == 0 {
			return true
		}
		for _, lp := range req.LocalPorts {
			if lp.Port == p {
				return true
			}
		}
		return false
	})

	ownedAvail := req.OwnedAvailable
	quickAvail := req.QuickAvailable
	// Defaults for manager leaves that only set the relevant side:
	// when neither flag was conceptually "configured", leaves set them explicitly.

	owned := &fakeProvider{
		id:      portforward.ProviderCloudflareOwned,
		avail:   ownedAvail,
		urlBase: "https://ephemeral-test.example.com",
	}
	quick := &fakeProvider{
		id:      portforward.ProviderCloudflareQuick,
		avail:   quickAvail,
		urlBase: "https://random-test.trycloudflare.com",
	}
	vm.RegisterProvider(owned)
	vm.RegisterProvider(quick)
	return vm, owned, quick, nil
}

func captureProvider(resp *Response, owned, quick *fakeProvider) {
	if owned != nil {
		resp.OwnedStopCount = owned.StopCount()
		if owned.LastStartPort() > 0 {
			resp.TunnelStartPort = owned.LastStartPort()
			resp.TunnelStartHostname = owned.LastHostname()
		}
	}
	if quick != nil {
		resp.QuickStopCount = quick.StopCount()
		if quick.LastStartPort() > 0 && resp.TunnelStartPort == 0 {
			resp.TunnelStartPort = quick.LastStartPort()
			resp.TunnelStartHostname = quick.LastHostname()
		}
		// Prefer the provider that actually started for hop asserts.
		if quick.LastStartPort() > 0 && owned != nil && owned.LastStartPort() == 0 {
			resp.TunnelStartPort = quick.LastStartPort()
			resp.TunnelStartHostname = quick.LastHostname()
		}
	}
}

func toVisitInfo(s *portforward.AdhocVisitSession) *VisitInfo {
	if s == nil {
		return nil
	}
	return &VisitInfo{
		ID:           s.ID,
		LocalPort:    s.LocalPort,
		ProxyPort:    s.ProxyPort,
		PublicURL:    s.PublicURL,
		Provider:     s.Provider,
		Hostname:     s.Hostname,
		IdleSeconds:  int(s.IdleTimeout / time.Second),
		Status:       s.Status,
		LastActivity: s.LastActivity,
		CreatedAt:    s.CreatedAt,
	}
}

func toVisitInfos(list []portforward.AdhocVisitSession) []VisitInfo {
	out := make([]VisitInfo, 0, len(list))
	for i := range list {
		info := toVisitInfo(&list[i])
		if info != nil {
			out = append(out, *info)
		}
	}
	return out
}

func localPortInfos(seeds []LocalPortSeed) []portforward.LocalPortInfo {
	out := make([]portforward.LocalPortInfo, 0, len(seeds))
	for _, s := range seeds {
		cmd := s.Command
		if cmd == "" {
			cmd = "app"
		}
		out = append(out, portforward.LocalPortInfo{
			Port:    s.Port,
			PID:     s.PID,
			PPID:    s.PPID,
			Command: cmd,
			Cmdline: s.Cmdline,
		})
	}
	return out
}

func withBearerAuth(validToken string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ping" || !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") || strings.TrimPrefix(authHeader, "Bearer ") != validToken {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func waitHTTPReady(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return nil
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	return fmt.Errorf("server not ready: %s", url)
}

func httpGetStatus(url string) (int, error) {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode, nil
}

func runAgentInProcess(argv []string) (int, string, string, error) {
	agentcliInProcessMu.Lock()
	defer agentcliInProcessMu.Unlock()

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		return 0, "", "", err
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		stdoutR.Close()
		stdoutW.Close()
		return 0, "", "", err
	}

	oldOut, oldErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = stdoutW, stderrW

	runErr := agentcli.Run(agentcli.RemoteProfile(), argv)

	_ = stdoutW.Close()
	_ = stderrW.Close()
	os.Stdout, os.Stderr = oldOut, oldErr

	outBytes, _ := io.ReadAll(stdoutR)
	errBytes, _ := io.ReadAll(stderrR)
	_ = stdoutR.Close()
	_ = stderrR.Close()

	stdout := string(outBytes)
	stderr := string(errBytes)
	exitCode := 0
	if runErr != nil {
		// Match product CLI: errors surface as "Error: …" on stderr with non-zero exit.
		if !strings.Contains(stderr, "Error:") {
			stderr += fmt.Sprintf("Error: %v\n", runErr)
		}
		exitCode = 1
	}
	return exitCode, stdout, stderr, nil
}

func writeMappingNames(path string, m map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func readMappingNames(path string) map[string]string {
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]string{}
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return map[string]string{}
	}
	if m == nil {
		return map[string]string{}
	}
	return m
}

// --- fake Provider ---

type fakeProvider struct {
	id      string
	avail   bool
	urlBase string

	mu       sync.Mutex
	stops    int
	startPort int
	hostname  string
}

func (p *fakeProvider) Name() string        { return p.id }
func (p *fakeProvider) DisplayName() string { return p.id }
func (p *fakeProvider) Description() string { return "fake " + p.id }
func (p *fakeProvider) Available() bool     { return p.avail }

func (p *fakeProvider) Start(port int, hostname string) (*portforward.TunnelHandle, error) {
	p.mu.Lock()
	p.startPort = port
	p.hostname = hostname
	p.mu.Unlock()

	url := p.urlBase
	if hostname != "" {
		// Owned-style: public URL uses the provided hostname.
		url = "https://" + hostname
	}
	ch := make(chan portforward.TunnelResult, 1)
	ch <- portforward.TunnelResult{PublicURL: url}
	return &portforward.TunnelHandle{
		Result: ch,
		Stop: func() {
			p.mu.Lock()
			p.stops++
			p.mu.Unlock()
		},
		Logs: portforward.NewLogBuffer(),
	}, nil
}

func (p *fakeProvider) StopCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stops
}

func (p *fakeProvider) LastStartPort() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.startPort
}

func (p *fakeProvider) LastHostname() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.hostname
}

// silence unused import if strconv not used in all builds
var _ = strconv.Itoa
var _ = bytes.MinRead
```
