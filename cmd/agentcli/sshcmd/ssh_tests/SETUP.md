# Scenario

**Feature**: remote-agent ssh P3 — ad-hoc SSH server, CryptoSSHRunner, relay compose, agentcli wire

```
# adhoc public-key SSH
ClientKeyPair -> AdhocServer(127.0.0.1:R) -> shell | remote command

# crypto runner
CryptoSSHRunner.Run(sess.LocalPort) -> x/crypto/ssh client -> SSH bytes

# compose through local relay
ServeService{Dial: TCP(Adhoc)} -> LocalRelay:L
CryptoSSHRunner -> :L -> Dial -> Adhoc :R

# CLI
operator -> agentcli.Run(["ssh", ...]) -> sshcmd defaults (store/runner/serve)
```

## Preconditions

- Package: `github.com/xhd2015/ai-critic/cmd/agentcli/sshcmd` (P1/P2 symbols exist;
  P3 symbols `AdhocServer`, `GenerateClientKeyPair`, `ClientKeyPair`,
  `CryptoSSHRunner`, `DialTCP` may be missing until implementer — suite is
  **RED** on compile/link).
- Package: `github.com/xhd2015/ai-critic/cmd/agentcli` — `Run` must gain `case "ssh"`.
- Parallel-safe: absolute paths from `d.DOCTEST_CASE`; ports via Listen `:0`;
  no `Setenv`/`Chdir`; no reassignment of process-global stdout for asserts.
- Prefer `golang.org/x/crypto/ssh` client/server (already in go.mod).
- P1 `tests/` (12) and P2 `session_tests/` (9) remain independent and GREEN.

## Steps

1. Root Setup defaults ProfileID, Root, ConfigDir, RemoteCommand, EchoNeedle.
2. Grouping Setup sets Scenario family (adhoc | crypto-runner | through-relay | agentcli).
3. Leaf Setup sets concrete Scenario and argv / needle overrides.
4. Root `Run` dispatches on Scenario and records Response fields.

## Context

- Module: `github.com/xhd2015/ai-critic`
- Tree home: `cmd/agentcli/sshcmd/ssh_tests` (sibling of P1 `tests/` and P2 `session_tests/`)
- L2 in-process only; no L3 system-ssh e2e.

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
	if req.RemoteCommand == "" {
		req.RemoteCommand = "echo hello"
	}
	if req.EchoNeedle == "" {
		req.EchoNeedle = "hello"
	}
	return nil
}
```
