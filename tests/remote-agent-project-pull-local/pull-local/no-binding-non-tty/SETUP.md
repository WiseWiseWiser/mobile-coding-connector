# Scenario

**Feature**: non-TTY pull-local requires binding or --local-path

```
# empty stdin pipe, no project_bindings, dirty remote
remote-agent project pull-local -> needs binding or interactive terminal
```

## Preconditions

Dirty remote; no seeded binding; `PipeStdin` true.

## Steps

1. Same-origin pair; dirty remote.
2. No `SeedBindings`; enable `PipeStdin`.
3. Args: `project pull-local pull-nobind-001`.

## Context

REQUIREMENT leaf `pull-local/no-binding-non-tty`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	pair := pairSameOriginRepos(t)
	remoteDir, localDir := pair.RemoteDir, pair.LocalDir
	dirtyTopLevelModifiedAndUntracked(t, remoteDir)
	registerPullProject(t, req, "pull-nobind-001", "pull-nobind", remoteDir)
	req.LocalPath = localDir
	req.PipeStdin = true
	req.Args = []string{"project", "pull-local", "pull-nobind-001"}
	return nil
}
```