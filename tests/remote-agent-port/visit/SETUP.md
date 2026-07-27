# Scenario

**Feature**: ad-hoc port visit (manager + CLI)

```
VisitSessionManager.Start / port visit -> public URL + idle proxy hop
```

## Preconditions

Fake owned/quick Providers; injectable listening checker and clock.

## Steps

1. Group leaves set provider availability and Op.
2. Run manager or CLI path.
3. Assert provider, session, idle, or CLI output.

## Context

Locked decisions: auto provider, 10m idle, warn-if-not-listening, no mapping-names write for owned ad-hoc.

```go
import (
	"testing"
	"time"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	if req.Port == 0 {
		req.Port = defaultTestPort
	}
	return nil
}
```
