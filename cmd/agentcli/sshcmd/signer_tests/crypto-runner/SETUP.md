# Scenario

**Feature**: CryptoSSHRunner Signer gate — wire-up required before dial

```
# nil Signer is a clear configuration error
CryptoSSHRunner{Signer:nil}.Run(Alive sess) -> error contains "signer"
```

## Preconditions

- Scenario family is crypto-runner (leaf sets concrete Scenario).
- Existing `CryptoSSHRunner` behavior: `"ssh signer not configured"`.

## Steps

1. Leaves set Scenario and any session fields needed for the gate.
2. Assert RunnerErr contains `signer`.

## Context

- Regression: proves production must set Signer (via EnsureClientKeyPair).

```go
import (
	"fmt"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	if req.ConfigDir == "" {
		return fmt.Errorf("crypto-runner: ConfigDir must be set by root Setup")
	}
	return nil
}
```
