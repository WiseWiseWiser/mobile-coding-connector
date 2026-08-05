# Scenario

**Feature**: remote-agent ssh P2 — session file store, local TCP relay, Serve Start(ctx)

```
# on-disk session
ServeService / tests -> FileSessionStore.Save/Load/Clear
  -> {Root}/ssh-sessions/{profileID}.json

# local relay splice
client -> LocalRelay(127.0.0.1:port) -> DialFunc() remote conn -> echo bytes

# serve lifecycle
ServeService.Start(ctx) -> listen + Save Alive + accept loop
cancel ctx -> Clear session + Close relay
```

## Preconditions

- Package: `github.com/xhd2015/ai-critic/cmd/agentcli/sshcmd` (P1 symbols exist;
  P2 symbols `FileSessionStore`, `LocalRelay`, `DialFunc`, `ServeService` may be
  missing until implementer — suite is **RED** on compile/link).
- Parallel-safe: absolute paths from `d.DOCTEST_CASE`; no `Setenv`/`Chdir`.
- Session path layout: `{Root}/ssh-sessions/{profileID}.json`.
- No live remote agent server; Dial always injected (or intentionally nil).
- P1 tree `cmd/agentcli/sshcmd/tests` remains independent and GREEN.

## Steps

1. Root Setup ensures `ProfileID` default `"default"`, allocates absolute
   `Root` and `ConfigDir` under `d.DOCTEST_CASE` when empty.
2. Grouping Setup sets Scenario family (session-store | local-relay | serve).
3. Leaf Setup sets concrete Scenario and any SessionToSave / payload.
4. Root `Run` dispatches on Scenario and records Response fields.

## Context

- Module: `github.com/xhd2015/ai-critic`
- Tree home: `cmd/agentcli/sshcmd/session_tests` (sibling of P1 `tests/`)
- L2 in-process only; no L3 e2e.

```go
import (
	"path/filepath"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	if req.ProfileID == "" {
		req.ProfileID = "default"
	}
	if req.Root == "" {
		req.Root = filepath.Join(d.DOCTEST_CASE, "store-root")
	}
	if req.ConfigDir == "" {
		req.ConfigDir = filepath.Join(d.DOCTEST_CASE, "session-config")
	}
	if req.EchoPayload == "" {
		req.EchoPayload = "hi"
	}
	return nil
}
```
