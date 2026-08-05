# Scenario

**Feature**: Full stack — ServeService Dial to Adhoc + CryptoSSHRunner via LocalPort

```
# compose
AdhocServer :R
ServeService{Dial: TCP(R)}.Start(ctx) -> LocalRelay :L + Alive session
CryptoSSHRunner -> :L -> Dial -> :R -> remote command
```

## Preconditions

- Scenario family: through-relay.
- Uses P2 ServeService/FileSessionStore + P3 Adhoc/CryptoSSHRunner.

## Steps

1. Start Adhoc with authorized key.
2. Start ServeService with Dial to Adhoc.Addr.
3. Wait Alive session; runner uses session LocalPort.
4. Optional cancel + teardown checks (serve-cancel leaf).

## Context

- Exit criterion for P3 L2: real SSH bytes through local relay.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	_ = req
	return nil
}
```
