# Scenario

**Feature**: API list returns seeded session metas

```
3 metas on disk -> GET /api/agent-run/sessions?limit=0 -> all three DTOs
```

## Preconditions

- Three ordered sessions seeded under store home.
- `limit=0` returns all.

## Steps

1. Seed three ordered sessions.
2. APILimit = 0.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setAPI(req)
	req.Seeds = seedThreeOrdered()
	req.APILimit = limitPtr(0)
	return nil
}
```
