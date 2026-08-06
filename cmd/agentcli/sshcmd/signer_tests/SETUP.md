# Scenario

**Feature**: remote-agent ssh Signer persist/reload and CryptoSSHRunner wire from disk

```
# ensure key material under configDir
EnsureClientKeyPair(configDir) -> id_ed25519 (0600) + ClientKeyPair

# reload stable identity
EnsureClientKeyPair(same dir) -> same Public material

# runner gate
CryptoSSHRunner{Signer:nil}.Run -> error contains "signer"

# compose from disk
Ensure -> Adhoc(authorized=Public) + ServeService Dial
CryptoSSHRunner{Signer: Ensure.Signer} -> echo signer-ok
```

## Preconditions

- Package: `github.com/xhd2015/ai-critic/cmd/agentcli/sshcmd`.
- **New** export `EnsureClientKeyPair` is **missing until implementer** — suite is
  classic **RED** (compile fail or Ensure errors) until wired.
- Existing: `ClientKeyPair`, `GenerateClientKeyPair`, `CryptoSSHRunner`,
  `AdhocServer`, `ServeService`, `FileSessionStore`, `DialTCP` (optional).
- Parallel-safe: absolute paths from `d.DOCTEST_CASE` / `d.DOCTEST_ROOT`; ports
  via Listen `:0`; no `Setenv`/`Chdir`; no process-global stdout reassignment.
- Sealed trees `tests/`, `session_tests/`, `ssh_tests/`, `tunnel_tests/` stay
  independent and GREEN.

## Steps

1. Root Setup defaults ProfileID, Root, ConfigDir, RemoteArgv, EchoNeedle.
2. Grouping Setup labels surface family (ensure-key-pair | crypto-runner | through-relay).
3. Leaf Setup sets concrete Scenario and leaf-specific argv/corrupt bytes.
4. Root `Run` dispatches on Scenario and fills Response fields.

## Context

- Module: `github.com/xhd2015/ai-critic`
- Tree home: `cmd/agentcli/sshcmd/signer_tests`
- L2 in-process only (library API + existing Adhoc/Serve compose).
- Private key path: `{ConfigDir}/id_ed25519`.

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
		req.ConfigDir = filepath.Join(d.DOCTEST_CASE, "ssh-config")
	}
	if len(req.RemoteArgv) == 0 {
		req.RemoteArgv = []string{"echo", "signer-ok"}
	}
	if req.EchoNeedle == "" {
		req.EchoNeedle = "signer-ok"
	}
	return nil
}
```
