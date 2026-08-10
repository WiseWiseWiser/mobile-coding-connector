# Scenario

**Feature**: remote-agent event-bus listen L2 harness

```
# L2: RunWithWriters (help/reject) | RunEventBusListen (injectable WS)
leaf Setup -> help argv OR hub/inject stream -> stdout/stderr Assert
```

## Preconditions

1. Product implements `remote-agent event-bus` (+ `listen`),
   `agentcli.RunWithWriters`, and `agentcli.RunEventBusListen` /
   `EventBusListenOpts` (classic TDD: missing → compile RED or assertion RED).
2. PHASE 2 `server/eventbus` Hub + `RegisterSubscribeWS` available for hub-mode
   leaves.
3. Shared Event types from `dot-pkgs/go-pkgs/eventbus` (module replace ok).
4. No product-binary e2e; no process env/cwd mutation.
5. Harness never reassigns `os.Stdout`/`os.Stderr`; all CLI and listen I/O uses
   injected `io.Writer` buffers.

## Steps

1. Root Setup defaults `Op` and common fixture fields.
2. Grouping Setup sets Op branch and dial mode defaults.
3. Leaf Setup fills Args or listen opts + event seeds.
4. Root `Run` executes CLI or `RunEventBusListen` and fills `Response`.
5. Leaf `Assert` checks exit code, help text, human/JSON lines, warnings.

## Context

Implements `/tmp/REQUIREMENT-DESIGN-PHASE-3-remote-agent-eventbus-listen.md`.
L2 mass only; no `e2e` label.

```go
import (
	"testing"
	"time"

	sharedeb "github.com/xhd2015/dot-pkgs/go-pkgs/eventbus"
	"github.com/xhd2015/doctest/session"
)

const (
	fixtureTypeSeatalk = sharedeb.TypeSeatalkMessageReceived
	fixtureTypeTTY     = sharedeb.TypeAgentTTYStarted
	fixtureSourceBot   = sharedeb.SourceSeatalkLocalBot
	fixtureSourceAgent = sharedeb.SourceAgentRun
	fixtureEventID1    = "evt-listen-001"
	fixtureEventID2    = "evt-listen-002"
	fixtureEventID3    = "evt-listen-003"
	fixtureTS1         = "2026-08-10T12:34:56.000Z"
	fixtureTS2         = "2026-08-10T12:35:01.000Z"
	fixtureTS3         = "2026-08-10T12:35:10.000Z"
	fixturePayload1    = `{"text":"hello-bus"}`
	fixturePayload2    = `{"text":"second"}`
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	if req == nil {
		return nil
	}
	if req.Op == "" {
		req.Op = "cli"
	}
	if req.FixedNow.IsZero() {
		req.FixedNow = time.Date(2026, 8, 10, 12, 34, 56, 0, time.Local)
	}
	return nil
}

func setCLI(req *Request, args ...string) {
	req.Op = "cli"
	req.Args = args
}

func setListenHub(req *Request) {
	req.Op = "listen"
	req.DialMode = "hub"
}

func seedOneLive(req *Request) {
	req.LiveEvents = []EventSeed{{
		ID:      fixtureEventID1,
		TS:      fixtureTS1,
		Source:  fixtureSourceBot,
		Type:    fixtureTypeSeatalk,
		Payload: fixturePayload1,
	}}
	if req.MaxEvents <= 0 {
		req.MaxEvents = 1
	}
}

func seedTwoLiveDifferentTypes(req *Request) {
	req.LiveEvents = []EventSeed{
		{
			ID:      fixtureEventID1,
			TS:      fixtureTS1,
			Source:  fixtureSourceBot,
			Type:    fixtureTypeSeatalk,
			Payload: fixturePayload1,
		},
		{
			ID:      fixtureEventID2,
			TS:      fixtureTS2,
			Source:  fixtureSourceAgent,
			Type:    fixtureTypeTTY,
			Payload: fixturePayload2,
		},
	}
}

func seedReplayThenLive(req *Request, n int) {
	req.Replay = n
	req.RecentEvents = []EventSeed{
		{
			ID:      fixtureEventID1,
			TS:      fixtureTS1,
			Source:  fixtureSourceBot,
			Type:    fixtureTypeSeatalk,
			Payload: `{"n":1}`,
		},
		{
			ID:      fixtureEventID2,
			TS:      fixtureTS2,
			Source:  fixtureSourceBot,
			Type:    fixtureTypeSeatalk,
			Payload: `{"n":2}`,
		},
	}
	req.LiveEvents = []EventSeed{{
		ID:      fixtureEventID3,
		TS:      fixtureTS3,
		Source:  fixtureSourceAgent,
		Type:    fixtureTypeTTY,
		Payload: `{"n":3}`,
	}}
	// Expect N recent + 1 live printed (when types unfiltered).
	req.MaxEvents = n + 1
}
```
