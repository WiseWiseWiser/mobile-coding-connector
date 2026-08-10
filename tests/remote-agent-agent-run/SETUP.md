# Scenario

**Feature**: remote-agent agent-run P1–P5 (full remote façade + local-only)

```
leaf Setup -> seed metas + injects
  -> sessions|attach|status|resume|send|msg|snapshot|watch|kill|run
  -> local-only rejects (focus|web|assets|pty)
  -> stdout / inject observation flags
```

## Preconditions

1. P1–P4 surfaces remain registered; ASSERTs sealed.
2. P4 injects as before; **P5** `Options.RunSession` inject for run.
3. Local-only commands fail in CLI without requiring server inject.
4. CLI via `agentcli.RunWithWriters`; store home constructor-injected.
5. No harness `t.Setenv` / `t.Chdir` / process stdio reassignment.

## Steps

1. Root Run creates temp home, seeds metas, wires injects.
2. Leaf sets Args and inject fields (or local-only command).
3. Assert exit codes, output markers, or inject observation.

## Context

Requirements: `/tmp/REQUIREMENT-DESIGN-PHASE-{1,2,3,4,5}-remote-agent-agent-run.md`.

```go
import (
	"fmt"
	"testing"
	"time"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	if req.Op == "" {
		req.Op = "cli"
	}
	return nil
}

func setCLI(req *Request, args ...string) {
	req.Op = "cli"
	req.Args = args
}

func setAPI(req *Request) {
	req.Op = "api"
}

func setAttach(req *Request, sessionID, ttyMode string) {
	req.Op = "attach"
	req.AttachSessionID = sessionID
	req.TTYMode = ttyMode
}

func setKeepalive(req *Request, sessionID string, pingEvery, wait time.Duration) {
	req.Op = "keepalive"
	req.AttachSessionID = sessionID
	req.TTYMode = "hold"
	req.PingInterval = pingEvery
	req.WaitForPing = wait
}

func limitPtr(n int) *int { return &n }

func boolPtr(v bool) *bool { return &v }

func seedN(n int) []SessionSeed {
	out := make([]SessionSeed, 0, n)
	for i := 0; i < n; i++ {
		year := 2030 - i
		ts := fmt.Sprintf("%04d-06-15T12:00:00Z", year)
		out = append(out, SessionSeed{
			SessionID: fmt.Sprintf("sess-%02d", i),
			Runner:    "grok",
			Status:    "finished",
			CreatedAt: ts,
			UpdatedAt: ts,
		})
	}
	return out
}

func seedThreeOrdered() []SessionSeed {
	return []SessionSeed{
		{SessionID: "sess-old", Runner: "codex", Status: "finished", CreatedAt: "2020-01-01T00:00:00Z", UpdatedAt: "2020-01-01T00:00:00Z"},
		{SessionID: "sess-mid", Runner: "grok", Status: "running", CreatedAt: "2021-06-01T00:00:00Z", UpdatedAt: "2021-06-01T00:00:00Z"},
		{SessionID: "sess-new", Runner: "opencode", Status: "idle", CreatedAt: "2022-12-01T00:00:00Z", UpdatedAt: "2022-12-01T00:00:00Z"},
	}
}

func seedAttachable(sessionID, termID string) []SessionSeed {
	return []SessionSeed{{
		SessionID: sessionID, Runner: "grok", Status: "running", TerminalSessionID: termID,
		CreatedAt: "2024-06-01T00:00:00Z", UpdatedAt: "2024-06-01T12:00:00Z",
	}}
}

func seedBoundExited(sessionID, runnerSess, termID string) []SessionSeed {
	return []SessionSeed{{
		SessionID: sessionID, Runner: "grok", Status: "finished",
		RunnerSessionID: runnerSess, TerminalSessionID: termID, Workspace: "/tmp/ws-resume",
		CreatedAt: "2024-06-01T00:00:00Z", UpdatedAt: "2024-06-02T00:00:00Z",
	}}
}

func seedUnboundFinished(sessionID string) []SessionSeed {
	return []SessionSeed{{
		SessionID: sessionID, Runner: "grok", Status: "finished",
		CreatedAt: "2024-06-01T00:00:00Z", UpdatedAt: "2024-06-01T00:00:00Z",
	}}
}

func seedLiveBound(sessionID, runnerSess string) []SessionSeed {
	return []SessionSeed{{
		SessionID: sessionID, Runner: "grok", Status: "running", RunnerSessionID: runnerSess,
		CreatedAt: "2024-06-01T00:00:00Z", UpdatedAt: "2024-06-01T12:00:00Z",
	}}
}

// seedLiveTTY is a running session with terminal_session_id for P4 live control.
func seedLiveTTY(sessionID, termID string) []SessionSeed {
	return seedAttachable(sessionID, termID)
}
```
