# Scenario

**Feature**: End-to-end client run with Signer loaded from disk via Ensure

```
# production-shaped compose (L2 loopback, no agent tunnel)
EnsureClientKeyPair(configDir) -> kp
AdhocServer.SetAuthorizedKeys(kp.Public); ServeService Dial=Adhoc
CryptoSSHRunner{Signer: Ensure(reload).Signer}.Run(echo) -> stdout needle
```

## Preconditions

- Scenario family is through-relay (leaf sets concrete Scenario).
- Reuses P2/P3 LocalRelay + Adhoc compose; does **not** set Signer from
  `GenerateClientKeyPair` alone — always via `EnsureClientKeyPair(configDir)`.

## Steps

1. Leaf sets Scenario + RemoteArgv + EchoNeedle.
2. Run: Ensure → Adhoc → Serve → Ensure reload → runner.Run.

## Context

- Closes the live-verify gap `ssh signer not configured` when key + Alive exist.

```go
import (
	"fmt"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	if req.ConfigDir == "" || req.Root == "" {
		return fmt.Errorf("through-relay: ConfigDir and Root must be set by root Setup")
	}
	return nil
}
```
