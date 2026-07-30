# Scenario

**Feature**: service add happy path with name, command, project-dir, working-dir

```
remote-agent service add --name demo-add --command "sleep 300" \
  --project-dir <tmp> --working-dir <tmp>
  -> exit 0, Created + name; services.json row; ListAll sees it
```

## Preconditions

1. Empty services.json (no pre-seed).
2. Product implements `service add` → POST create.

## Steps

1. Create temp project and working dirs.
2. Run `service add` with required flags.
3. Assert Created, disk, ListAll.

## Context

REQUIREMENT leaf: `add/happy-path`. Default is definition-only (no start).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	proj := t.TempDir()
	work := t.TempDir()
	req.LocalProjectDir = proj
	req.TargetName = "demo-add"
	setCLI(req,
		"service", "add",
		"--name", "demo-add",
		"--command", "sleep 300",
		"--project-dir", proj,
		"--working-dir", work,
	)
	return nil
}
```
