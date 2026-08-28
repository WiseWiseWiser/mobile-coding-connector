# Scenario

**Feature**: GET /api/grok/usage returns ready JSON

```
server fetch (AI_CRITIC_GROK_USAGE_FIXTURE) -> GET :23712/api/grok/usage -> status ready
```

## Preconditions

`testdata/usage-ready.json` exported as `AI_CRITIC_GROK_USAGE_FIXTURE` in daemon env.

## Steps

1. `MockScript=usage-ready.json`, `WaitAPIReadySecs=15`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.MockScript = "usage-ready.json"
	req.WaitAPIReadySecs = 15
	return nil
}
```
