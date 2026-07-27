# Scenario

**Feature**: local-agent `bookmarks` CLI against HTTP API + isolated HOME

```
# agentcli.Run(LocalProfile()) --server httptest --token
bookmarks list|add|add-folder|set|delete|move|open|-h
```

## Preconditions

1. Mode `cli`.
2. Run mounts bookmarks API on httptest and sets testhooks home override.
3. CLIArgs are subcommand argv after global flags (include leading `bookmarks`).

## Steps

1. Leaf sets CLIArgs (+ optional SeedAdds / CLIEnv).
2. Run captures stdout/stderr/exit and reloads Doc from store path.
3. Assert output and/or tree.

## Context

User-facing CRUD + open dry-run.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Mode = "cli"
	return nil
}
```
