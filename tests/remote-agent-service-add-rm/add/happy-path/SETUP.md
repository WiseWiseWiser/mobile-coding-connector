# Scenario

**Feature**: service add happy path with name, command, working-dir

```
remote-agent service add --name demo-add --command "sleep 300" \
  --working-dir <tmp>
  -> exit 0, Created + name; services.json row; List sees it
```

## Preconditions

1. Empty services.json (no pre-seed).
2. Product implements `service add` → POST create.

## Steps

1. Create temp working dir.
2. Run `service add` with required flags.
3. Assert Created, disk, List.

## Context

REQUIREMENT leaf: `add/happy-path`. Default is definition-only (no start).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	work := t.TempDir()
	req.LocalWorkingDir = work
	req.TargetName = "demo-add"
	setCLI(req,
		"service", "add",
		"--name", "demo-add",
		"--command", "sleep 300",
		"--working-dir", work,
	)
	return nil
}
```
