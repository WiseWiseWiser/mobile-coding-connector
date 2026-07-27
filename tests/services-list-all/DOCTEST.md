# Services List All API Doctests

L2 library tests for `services.Manager.List` / `ListAll` project scoping
(`?all=1` bypass). No product binary.

# DSN (Domain Specific Notion)

Most leaves are **L2 in-process**: `services.NewManagerFromDefinitions` +
`List(projectDir)` / `ListAll()` (no `ai-critic-server` binary). Zero e2e smokes
— pure Manager filtering is Parallel-safe.

**Participants**

- **L2: services.Manager** — in-memory definitions via `NewManagerFromDefinitions`.
- **Service definitions** — rows with optional `projectDir` scoping.
- **List vs ListAll** — `List(projectDir)` filters by project; `ListAll` returns all.

**Behaviors**

- Default list returns only services matching the given project scope.
- ListAll returns services across all `projectDir` values.
- Status objects expose `id` fields used by API responses.

## Version

0.0.3

## Decision Tree

```
[services list API]
 |
 +-- list-scoped-default/             (LEAF)   List(project) project-scoped only
 +-- list-all/                        (LEAF)   ListAll returns cross-scope services
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `list-scoped-default` | Default list excludes other-project services |
| 2 | `list-all` | ListAll includes services from all project dirs |

## Parameter Coverage

| Leaf | Query | Seeded projects | Expect |
|------|-------|-----------------|--------|
| list-scoped-default | none | local + other | only local ID |
| list-all | `all=1` | local + other | both IDs |

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
	"path/filepath"
	"testing"

	"github.com/xhd2015/ai-critic/server/services"
	"github.com/xhd2015/doctest/session"
)

type ServiceSeed struct {
	ID         string
	Name       string
	Command    string
	ProjectDir string
}

type Request struct {
	Op string // list-scoped | list-all

	LocalProjectDir string
	OtherProjectDir string
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

	localDir := req.LocalProjectDir
	if localDir == "" {
		localDir = configHome
	}
	otherDir := req.OtherProjectDir
	if otherDir == "" {
		otherDir = filepath.Join(configHome, "other-project")
	}
	if err := os.MkdirAll(otherDir, 0755); err != nil {
		return nil, err
	}

	defs := []services.ServiceDefinition{
		{ID: req.LocalServiceID, Name: "web", Command: "sleep 300", ProjectDir: localDir, CreatedAt: "2026-07-07T00:00:00Z", UpdatedAt: "2026-07-07T00:00:00Z"},
		{ID: req.OtherServiceID, Name: "api", Command: "sleep 300", ProjectDir: otherDir, CreatedAt: "2026-07-07T00:00:00Z", UpdatedAt: "2026-07-07T00:00:00Z"},
	}
	m := services.NewManagerFromDefinitions(defs)

	var listed []services.ServiceStatus
	switch req.Op {
	case "list-scoped":
		listed = m.List(localDir)
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
