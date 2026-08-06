# Scenario

**Feature**: CryptoSSHRunner with nil Signer errors before remote work

```
# gate
runner.Signer = nil; sess.LocalPort > 0; Alive
  -> Run returns error containing "signer"
```

## Preconditions

- Scenario: `crypto-nil-signer`.
- Session has LocalPort > 0 so check order hits Signer before invalid-port.
- No AdhocServer required (error before dial).

## Steps

1. Build CryptoSSHRunner with Signer nil and InsecureIgnoreHostKey true.
2. Run with remote argv; capture RunnerErr.

## Context

- Existing message: `ssh signer not configured`. Assert substring `signer`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Scenario = ScenarioCryptoNilSigner
	return nil
}
```
