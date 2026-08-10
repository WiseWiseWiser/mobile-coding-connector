# ai-critic server event-bus (hub + publish loopback + WS subscribe)

L2 in-process doctests for the injectable package
`github.com/xhd2015/ai-critic/server/eventbus` (PHASE 2 hub + listeners).

Classic TDD: product package is greenfield — leaves stay **RED** until implementer
ships Hub / PublishServer / RegisterSubscribeWS / flag helpers.

Shared wire types come from `github.com/xhd2015/dot-pkgs/go-pkgs/eventbus`
(PHASE 1). Implementer may add a `go.mod` `replace` to the brought tree.

# DSN (Domain Specific Notion)

In-process event hub with a loopback HTTP publish listener and a main-mux
WebSocket subscribe path; keep-alive forwards related server flags to the child.

**Participants**

- **Hub** — in-memory ring (~200) + fan-out; `Publish` fills missing `id`/`ts`.
- **PublishServer** — loopback-only HTTP `POST /publish` (default port constant
  23891); optional Bearer token; hard-fails if port already in use.
- **RegisterSubscribeWS** — main mux `GET /api/event-bus/ws` streams JSON Events.
- **PublishConfig helpers** — `DefaultPublishPort`, `ResolvePublishConfig`,
  `AppendEventBusServerArgs` for keep-alive child argv.
- **Shared Event** — `dot-pkgs/go-pkgs/eventbus` JSON envelope.
- **Test harness** — real `127.0.0.1` listeners (`:0` where possible); no product
  binary e2e.

**Behaviors**

- Hub assigns non-empty `id`/`ts` when missing; preserves provided values; ring
  keeps last N (default design size 200); all live subscribers receive each event.
- Publish HTTP is open when no token configured; with token, missing/wrong Bearer
  → 401; correct Bearer → accepted.
- Publish binds loopback only; second bind on the same port fails.
- WS subscribers on `/api/event-bus/ws` receive events from Hub.Publish and from
  HTTP `/publish`.
- Flag resolve: zero port → default 23891; `--no-event-bus-publish` disables.
- `AppendEventBusServerArgs` forwards non-default port/token and no-publish.

## Version

0.0.2

## Decision Tree

```
event-bus/
 |
 +-- hub/                                    (GROUP) in-memory Hub
 |    +-- publish/                           (GROUP) Publish enrich / preserve
 |    |    +-- assigns-id-ts-when-missing/   (LEAF)
 |    |    +-- preserves-provided-id-ts/     (LEAF)
 |    +-- ring/                              (GROUP) ring buffer capacity
 |    |    +-- keeps-last-200/               (LEAF)
 |    +-- fanout/                            (GROUP) live subscribers
 |         +-- two-subscribers/              (LEAF)
 |
 +-- publish-server/                         (GROUP) loopback HTTP publish
 |    +-- auth/                              (GROUP) token policy
 |    |    +-- open-when-no-token/           (LEAF) open publish
 |    |    +-- bearer-ok/                    (LEAF) correct Bearer accepted
 |    |    +-- missing-bearer-401/           (LEAF)
 |    |    +-- wrong-bearer-401/             (LEAF)
 |    +-- bind/                              (GROUP) address binding
 |    |    +-- loopback-127/                 (LEAF) 127.0.0.1 only
 |    +-- conflict/                          (GROUP) port contention
 |         +-- port-in-use-fails/            (LEAF)
 |
 +-- subscribe-ws/                           (GROUP) main mux WS path
 |    +-- from-hub-publish/                  (LEAF) Hub.Publish → WS
 |    +-- from-http-publish/                 (LEAF) POST /publish → WS
 |
 +-- config/                                 (GROUP) flags + keep-alive argv
      +-- resolve/                           (GROUP) ResolvePublishConfig
      |    +-- default-port-23891/           (LEAF)
      |    +-- no-publish-disables/          (LEAF)
      +-- append-args/                       (GROUP) AppendEventBusServerArgs
           +-- non-default-port-token/       (LEAF)
           +-- no-publish-flag/              (LEAF)
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `hub/publish/assigns-id-ts-when-missing` | Empty id/ts filled by `Hub.Publish` |
| 2 | `hub/publish/preserves-provided-id-ts` | Non-empty id/ts left unchanged |
| 3 | `hub/ring/keeps-last-200` | After 250 publishes on ring 200, `Recent` keeps last 200 |
| 4 | `hub/fanout/two-subscribers` | Two `Subscribe` channels both receive the same event |
| 5 | `publish-server/auth/open-when-no-token` | No server token → POST /publish succeeds |
| 6 | `publish-server/auth/bearer-ok` | Token configured + correct Bearer → success |
| 7 | `publish-server/auth/missing-bearer-401` | Token configured, no Authorization → 401 |
| 8 | `publish-server/auth/wrong-bearer-401` | Token configured, wrong Bearer → 401 |
| 9 | `publish-server/bind/loopback-127` | Server listens on 127.0.0.1 (loopback only) |
| 10 | `publish-server/conflict/port-in-use-fails` | Second `StartPublishServer` on same port fails |
| 11 | `subscribe-ws/from-hub-publish` | WS client receives event after `Hub.Publish` |
| 12 | `subscribe-ws/from-http-publish` | WS client receives event after HTTP POST /publish |
| 13 | `config/resolve/default-port-23891` | `DefaultPublishPort()==23891`; resolve zero port → 23891 |
| 14 | `config/resolve/no-publish-disables` | `noPublish=true` → `Disabled` |
| 15 | `config/append-args/non-default-port-token` | Child argv gains port + token flags |
| 16 | `config/append-args/no-publish-flag` | Child argv gains `--no-event-bus-publish` |

## Parameter Coverage

| Leaf | Op | Key inputs | Expected |
|------|-----|------------|----------|
| assigns-id-ts-when-missing | hub-publish | empty id/ts | returned id/ts non-empty |
| preserves-provided-id-ts | hub-publish | fixed id/ts | same id/ts |
| keeps-last-200 | hub-publish | ring=200, N=250 | Recent len 200, newest last |
| two-subscribers | hub-subscribe | 2 subs | both receive |
| open-when-no-token | publish-http | Token="" | 2xx |
| bearer-ok | publish-http | Token+Bearer match | 2xx |
| missing-bearer-401 | publish-http | Token set, no auth | 401 |
| wrong-bearer-401 | publish-http | wrong Bearer | 401 |
| loopback-127 | publish-bind | 127.0.0.1:0 | Addr is loopback |
| port-in-use-fails | publish-port-in-use | same port twice | 2nd err |
| from-hub-publish | subscribe-ws | Hub.Publish | WS Event |
| from-http-publish | subscribe-ws | POST /publish | WS Event |
| default-port-23891 | resolve-config | portFlag=0 | Port=23891 |
| no-publish-disables | resolve-config | noPublish | Disabled |
| non-default-port-token | append-args | port+token | flags present |
| no-publish-flag | append-args | Disabled | --no-event-bus-publish |

## How to Run

From ai-critic module root:

```sh
doctest vet ./tests/event-bus
doctest test ./tests/event-bus/...
```

Single leaf:

```sh
doctest test ./tests/event-bus/hub/publish/assigns-id-ts-when-missing
doctest test ./tests/event-bus/publish-server/auth/open-when-no-token
```

```go
import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/xhd2015/ai-critic/server/eventbus"
	sharedeb "github.com/xhd2015/dot-pkgs/go-pkgs/eventbus"
	"github.com/xhd2015/doctest/session"
)

// Request drives one L2 scenario against server/eventbus.
// Op selects the public surface under test.
type Request struct {
	Op string // hub-publish | hub-subscribe | publish-http | publish-bind | publish-port-in-use | subscribe-ws | resolve-config | append-args | default-port

	// Hub
	RingSize        int
	Event           sharedeb.Event // single event for publish / WS fixtures
	PublishCount    int            // when >0, publish N generated events (ring tests)
	RecentN         int
	SubscriberCount int

	// PublishServer
	ServerToken string // PublishServerOpts.Token
	ClientToken string // Authorization Bearer for client POST (empty + OmitAuth = no header)
	OmitAuth    bool
	ListenAddr  string // e.g. "127.0.0.1:0"; empty → "127.0.0.1:0"

	// subscribe-ws
	// PublishVia: "hub" | "http"
	PublishVia string

	// Config flags
	PortFlag  int
	TokenFlag string
	NoPublish bool
	BaseArgs  []string
}

// Response holds observed outputs for Assert.
type Response struct {
	// Hub
	Published  sharedeb.Event
	Recent     []sharedeb.Event
	Received   [][]sharedeb.Event // per-subscriber receive lists

	// HTTP publish
	StatusCode int
	Body       string

	// Bind / server
	ListenAddr   string // actual bound address host:port
	IsLoopback   bool
	SecondErr    error  // port-in-use second Start error
	FirstStarted bool

	// WS
	WSEvents []sharedeb.Event

	// Config
	DefaultPort int
	Config      eventbus.PublishConfig
	ArgsOut     []string
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	t.Helper()
	resp := &Response{}

	switch req.Op {
	case "default-port":
		resp.DefaultPort = eventbus.DefaultPublishPort()
		return resp, nil

	case "resolve-config":
		resp.Config = eventbus.ResolvePublishConfig(req.PortFlag, req.TokenFlag, req.NoPublish)
		resp.DefaultPort = eventbus.DefaultPublishPort()
		return resp, nil

	case "append-args":
		cfg := eventbus.ResolvePublishConfig(req.PortFlag, req.TokenFlag, req.NoPublish)
		resp.Config = cfg
		base := append([]string(nil), req.BaseArgs...)
		resp.ArgsOut = eventbus.AppendEventBusServerArgs(base, cfg)
		return resp, nil

	case "hub-publish":
		return runHubPublish(t, req)

	case "hub-subscribe":
		return runHubSubscribe(t, req)

	case "publish-http":
		return runPublishHTTP(t, req)

	case "publish-bind":
		return runPublishBind(t, req)

	case "publish-port-in-use":
		return runPublishPortInUse(t, req)

	case "subscribe-ws":
		return runSubscribeWS(t, req)

	default:
		return nil, fmt.Errorf("unknown Op %q", req.Op)
	}
}

func runHubPublish(t *testing.T, req *Request) (*Response, error) {
	t.Helper()
	ring := req.RingSize
	if ring <= 0 {
		ring = 200
	}
	hub := eventbus.NewHub(ring)
	resp := &Response{}

	if req.PublishCount > 0 {
		for i := 0; i < req.PublishCount; i++ {
			ev := sharedeb.Event{
				Source:  sharedeb.SourceAgentRun,
				Type:    sharedeb.TypeAgentTTYStarted,
				Payload: json.RawMessage(fmt.Sprintf(`{"n":%d}`, i)),
			}
			// encode sequence in id for "preserves" style ring checks when set by hub
			out := hub.Publish(ev)
			resp.Published = out
		}
		n := req.RecentN
		if n <= 0 {
			n = ring
		}
		resp.Recent = hub.Recent(n)
		return resp, nil
	}

	out := hub.Publish(req.Event)
	resp.Published = out
	if req.RecentN > 0 {
		resp.Recent = hub.Recent(req.RecentN)
	} else {
		resp.Recent = hub.Recent(1)
	}
	return resp, nil
}

func runHubSubscribe(t *testing.T, req *Request) (*Response, error) {
	t.Helper()
	hub := eventbus.NewHub(200)
	n := req.SubscriberCount
	if n <= 0 {
		n = 2
	}
	type sub struct {
		ch     <-chan sharedeb.Event
		cancel func()
	}
	subs := make([]sub, 0, n)
	for i := 0; i < n; i++ {
		ch, cancel := hub.Subscribe()
		subs = append(subs, sub{ch: ch, cancel: cancel})
	}
	defer func() {
		for _, s := range subs {
			s.cancel()
		}
	}()

	out := hub.Publish(req.Event)
	resp := &Response{Published: out, Received: make([][]sharedeb.Event, n)}
	deadline := time.After(2 * time.Second)
	for i, s := range subs {
		select {
		case ev, ok := <-s.ch:
			if !ok {
				return resp, fmt.Errorf("subscriber %d channel closed", i)
			}
			resp.Received[i] = append(resp.Received[i], ev)
		case <-deadline:
			return resp, fmt.Errorf("timeout waiting for subscriber %d", i)
		}
	}
	return resp, nil
}

func publishServerOpts(req *Request) eventbus.PublishServerOpts {
	return eventbus.PublishServerOpts{Token: req.ServerToken}
}

func startPublish(t *testing.T, hub *eventbus.Hub, req *Request) *eventbus.PublishServer {
	t.Helper()
	addr := req.ListenAddr
	if addr == "" {
		addr = "127.0.0.1:0"
	}
	srv, err := eventbus.StartPublishServer(addr, hub, publishServerOpts(req))
	if err != nil {
		t.Fatalf("StartPublishServer: %v", err)
	}
	t.Cleanup(func() { _ = srv.Close() })
	return srv
}

func runPublishHTTP(t *testing.T, req *Request) (*Response, error) {
	t.Helper()
	hub := eventbus.NewHub(200)
	srv := startPublish(t, hub, req)
	url := "http://" + srv.Addr() + "/publish"
	body, err := json.Marshal(req.Event)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if !req.OmitAuth && req.ClientToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+req.ClientToken)
	}
	client := &http.Client{Timeout: 3 * time.Second}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()
	b, _ := io.ReadAll(httpResp.Body)
	return &Response{
		StatusCode: httpResp.StatusCode,
		Body:       string(b),
		ListenAddr: srv.Addr(),
		Recent:     hub.Recent(1),
	}, nil
}

func runPublishBind(t *testing.T, req *Request) (*Response, error) {
	t.Helper()
	hub := eventbus.NewHub(200)
	srv := startPublish(t, hub, req)
	addr := srv.Addr()
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		// Addr might already be host:port or include scheme — try parse
		return &Response{ListenAddr: addr, IsLoopback: false}, fmt.Errorf("SplitHostPort(%q): %w", addr, err)
	}
	ip := net.ParseIP(host)
	return &Response{
		ListenAddr: addr,
		IsLoopback: ip != nil && ip.IsLoopback(),
	}, nil
}

func runPublishPortInUse(t *testing.T, req *Request) (*Response, error) {
	t.Helper()
	hub1 := eventbus.NewHub(200)
	// Bind an explicit free port on loopback first.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	srv1, err := eventbus.StartPublishServer(addr, hub1, publishServerOpts(req))
	if err != nil {
		return &Response{FirstStarted: false}, err
	}
	t.Cleanup(func() { _ = srv1.Close() })

	hub2 := eventbus.NewHub(200)
	srv2, err2 := eventbus.StartPublishServer(addr, hub2, publishServerOpts(req))
	if err2 == nil && srv2 != nil {
		t.Cleanup(func() { _ = srv2.Close() })
	}
	return &Response{
		FirstStarted: true,
		ListenAddr:   addr,
		SecondErr:    err2,
	}, nil
}

func runSubscribeWS(t *testing.T, req *Request) (*Response, error) {
	t.Helper()
	hub := eventbus.NewHub(200)
	mux := http.NewServeMux()
	eventbus.RegisterSubscribeWS(mux, hub)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	httpSrv := &http.Server{Handler: mux}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(ctx)
		_ = ln.Close()
	})
	go func() { _ = httpSrv.Serve(ln) }()

	wsURL := "ws://" + ln.Addr().String() + "/api/event-bus/ws"
	dialer := websocket.Dialer{HandshakeTimeout: 2 * time.Second}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("ws dial: %w", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	// Optional publish server for HTTP path
	var pub *eventbus.PublishServer
	if req.PublishVia == "http" {
		pub = startPublish(t, hub, req)
	}

	// Give the server a moment to register the subscriber
	time.Sleep(50 * time.Millisecond)

	switch req.PublishVia {
	case "hub", "":
		hub.Publish(req.Event)
	case "http":
		if pub == nil {
			return nil, fmt.Errorf("publish via http requires publish server")
		}
		body, err := json.Marshal(req.Event)
		if err != nil {
			return nil, err
		}
		url := "http://" + pub.Addr() + "/publish"
		httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Content-Type", "application/json")
		if req.ServerToken != "" {
			httpReq.Header.Set("Authorization", "Bearer "+req.ServerToken)
		}
		client := &http.Client{Timeout: 3 * time.Second}
		httpResp, err := client.Do(httpReq)
		if err != nil {
			return nil, err
		}
		defer httpResp.Body.Close()
		if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
			b, _ := io.ReadAll(httpResp.Body)
			return nil, fmt.Errorf("publish status %d: %s", httpResp.StatusCode, b)
		}
	default:
		return nil, fmt.Errorf("unknown PublishVia %q", req.PublishVia)
	}

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := conn.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("ws read: %w", err)
	}
	var ev sharedeb.Event
	if err := json.Unmarshal(data, &ev); err != nil {
		return nil, err
	}
	return &Response{WSEvents: []sharedeb.Event{ev}}, nil
}
```
