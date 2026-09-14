# Services List API Doctests

L2 library tests for `services.Manager.List` / `ListAll` (global list; no
project scope). No product binary.

# DSN (Domain Specific Notion)

Most leaves are **L2 in-process**: `services.NewManagerFromDefinitions` +
`List()` / `ListAll()` (no `ai-critic-server` binary). Zero e2e smokes —
pure Manager listing is Parallel-safe.

**Participants**

- **L2: services.Manager** — in-memory definitions via `NewManagerFromDefinitions`.
- **Service definitions** — global rows (no projectDir).
- **List / ListAll** — both return every service; ListAll is an alias of List.

**Behaviors**

- Default list returns every managed service.
- ListAll returns the same full set.
- Status objects expose `id` fields used by API responses.

## Version

0.0.4

## Decision Tree

```
[services list API]
 |
 +-- list-scoped-default/             (LEAF)   List() returns all services
 +-- list-all/                        (LEAF)   ListAll returns all services
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `list-scoped-default` | List includes every seeded service |
| 2 | `list-all` | ListAll includes every seeded service |

## Parameter Coverage

| Leaf | Op | Seeded services | Expect |
|------|----|-----------------|--------|
| list-scoped-default | list | web + api | both IDs |
| list-all | list-all | web + api | both IDs |

## How to Run

```sh
doctest vet ./tests/services-list-all
doctest test ./tests/services-list-all/...
doctest test --label e2e ./tests/services-list-all/...  # 0 smokes
```

```go
import (
	"fmt"
	"os"
	"testing"

	"github.com/xhd2015/ai-critic/server/services"
	"github.com/xhd2015/doctest/session"
)

type ServiceSeed struct {
	ID      string
	Name    string
	Command string
}

type Request struct {
	Op string // list | list-all

	LocalServiceID  string
	OtherServiceID  string

	// UseCLI / E2E reserved; this tree is pure L2 (no binary path).
	UseCLI bool
	E2E    bool
}

type Response struct {
	ConfigHome string
	ListedIDs  []string
	HTTPStatus int
	Body       string
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	resp := &Response{HTTPStatus: 200}
	if req.LocalServiceID == "" {
		req.LocalServiceID = "local-web"
	}
	if req.OtherServiceID == "" {
		req.OtherServiceID = "other-api"
	}

	configHome, err := os.MkdirTemp("", "services-list-all-*")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(configHome) })
	resp.ConfigHome = configHome

	defs := []services.ServiceDefinition{
		{ID: req.LocalServiceID, Name: "web", Command: "sleep 300", CreatedAt: "2026-07-07T00:00:00Z", UpdatedAt: "2026-07-07T00:00:00Z"},
		{ID: req.OtherServiceID, Name: "api", Command: "sleep 300", CreatedAt: "2026-07-07T00:00:00Z", UpdatedAt: "2026-07-07T00:00:00Z"},
	}
	m := services.NewManagerFromDefinitions(defs)

	var listed []services.ServiceStatus
	switch req.Op {
	case "list", "list-scoped":
		listed = m.List()
	case "list-all":
		listed = m.ListAll()
	default:
		return nil, fmt.Errorf("unknown op %q", req.Op)
	}

	ids := make([]string, 0, len(listed))
	for _, svc := range listed {
		ids = append(ids, svc.ID)
	}
	resp.ListedIDs = ids
	return resp, nil
}
```
