# Scenario

**Feature**: API ?limit=1 caps the list

```
3 metas -> GET /api/agent-run/sessions?limit=1 -> one session (newest)
```

## Preconditions

- Three ordered sessions on disk.

## Steps

1. Seed three ordered sessions.
2. APILimit = 1.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setAPI(req)
	req.Seeds = seedThreeOrdered()
	req.APILimit = limitPtr(1)
	return nil
}
```
