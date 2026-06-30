# Scenario

**Feature**: `project list` and `git-config get` show bound local directory

```
# isolated HOME + optional project_bindings -> printProjectGitConfig -> Local Dir line
remote-agent project list|git-config get -> stdout with Local Dir after Dir
```

## Preconditions

- Root `Run` writes `remote-agent-config.json` under isolated agent `HOME`.
- Binding lookup uses normalized `--server` and registered project `Dir`.

## Steps

1. Descendant leaves seed git repos and optional `SeedBindings` on `Request`.
2. Leaves set `req.Args` for `project list`, `project list --dirty`, or `project git-config get`.

## Context

REQUIREMENT: project list Local Dir + git-config get via `printProjectGitConfig`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if len(req.Args) == 0 {
		req.Args = []string{"project", "list"}
	}
	return nil
}
```