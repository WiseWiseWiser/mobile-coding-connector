# Scenario

**Feature**: Alive session + key on disk → CryptoSSHRunner Signer from Ensure succeeds

```
# full path without manually inventing a key outside Ensure
Ensure(configDir) -> authorize Adhoc + Serve Alive
Ensure(configDir) again -> runner.Signer
runner.Run(["echo","signer-ok"]) -> stdout contains signer-ok
```

## Preconditions

- Scenario: `through-relay-load-from-disk`.
- RemoteArgv `["echo","signer-ok"]`; EchoNeedle `signer-ok`.
- Signer for runner comes from second EnsureClientKeyPair (disk), not a free
  `GenerateClientKeyPair` only used in the harness for adhoc auth without Ensure.

## Steps

1. EnsureClientKeyPair creates/loads key under ConfigDir.
2. Start Adhoc authorized with that Public; ServeService Dial to Adhoc.
3. Wait for Alive session; runner uses Signer from Ensure reload.
4. Assert stdout needle; no "signer not configured".

## Context

- L2 in-process compose equivalent to production wire after Ensure exists.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Scenario = ScenarioRelayLoadFromDisk
	req.RemoteArgv = []string{"echo", "signer-ok"}
	req.EchoNeedle = "signer-ok"
	return nil
}
```
