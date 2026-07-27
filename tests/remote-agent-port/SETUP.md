# Scenario

**Feature**: remote-agent port list + ad-hoc visit L2 harness

```
# L2: VisitSessionManager + fake Providers + agentcli.Run
leaf Setup -> seed ports/providers -> port list|visit OR manager.Start
       -> stdout/stderr OR session fields + Sweep
```

## Preconditions

1. Product implements `portforward.VisitSessionManager` and `remote-agent port`
   (classic TDD: missing → compile RED or assertion RED).
2. Each leaf gets an isolated temp `configHome` and mapping-names path.
3. Fake Providers implement `portforward.Provider`; no live cloudflared.
4. CLI leaves use bearer token `lib.TestPassword` against the L2 mux.
5. Idle leaves inject `SetNow` + `Sweep` (no real 10-minute sleeps).

## Steps

1. Root `Run` creates temp config home and mapping-names path.
2. Leaf `Setup` sets `Op`, ports, provider availability, idle, and CLI `Args`.
3. `Run` either drives `VisitSessionManager` or `agentcli.Run` against the L2 mux.
4. Leaf `Assert` checks exit code, output templates, session fields, or file side effects.

## Context

Implements `/tmp/REQUIREMENT-DESIGN-remote-agent-port.md`. Prefer L2 mass; no e2e label.

```go
import (
	"testing"
	"time"

	"github.com/xhd2015/doctest/session"
)

const (
	defaultTestPort     = 18080
	defaultIdleShort    = 2 * time.Second
	defaultIdleTenMin   = 10 * time.Minute
	seedListenerPort    = 3000
	seedListenerPID     = 4242
	seedListenerCommand = "node"
	seedForwardPort     = 4000
	seedForwardURL      = "https://persist.example.com"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	if req.Op == "" {
		req.Op = "cli"
	}
	return nil
}

// setCLI configures a CLI leaf.
func setCLI(req *Request, args ...string) {
	req.Op = "cli"
	req.Args = args
}

// setManagerDefaults enables both fake providers unless a leaf already set them.
// Leaves must set OwnedAvailable / QuickAvailable explicitly for provider tests.
func enableOwnedQuick(req *Request, owned, quick bool) {
	req.OwnedAvailable = owned
	req.QuickAvailable = quick
}

func seedOneListener(req *Request) {
	req.LocalPorts = []LocalPortSeed{{
		Port:    seedListenerPort,
		PID:     seedListenerPID,
		PPID:    1,
		Command: seedListenerCommand,
		Cmdline: seedListenerCommand + " server.js",
	}}
}

func seedOneForward(req *Request) {
	req.Forwards = []ForwardSeed{{
		LocalPort: seedForwardPort,
		Label:     "persist-app",
		PublicURL: seedForwardURL,
		Status:    "active",
		Provider:  "cloudflare_owned",
	}}
}
```
