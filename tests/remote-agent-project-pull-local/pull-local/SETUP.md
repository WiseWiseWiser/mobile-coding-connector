# Scenario

**Feature**: `remote-agent project pull-local` transfers dirty remote state locally

```
# binding / flags / submodule checks -> worktree under project-worktrees -> optional remote truncate
remote-agent project pull-local <target> [flags] -> local worktree + remote porcelain state
```

## Preconditions

Remote project registered; local repo shares origin when pull should succeed.

## Steps

1. Leaf seeds remote cleanliness, bindings, submodules, or flags on `Request`.
2. `Run` may pipe stdin or run two pull-local invocations for worktree suffix tests.
3. `Assert` checks worktree tree, file contents, remote `git status --porcelain`, and output.

## Context

Grouping node for pull-local: success path, guard rails, flags, submodules.

```go
import (
	"os"
	"path/filepath"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	if len(req.Args) >= 2 && req.Args[1] != "pull-local" {
		t.Fatalf("pull-local group: unexpected subcommand argv %v", req.Args)
	}
	return nil
}

// pairSameOriginWithSubmodule returns remote/local clones whose parent repo contains
// a clean initialized submodule at `submod/`.
func pairSameOriginWithSubmodule(t *testing.T) RepoPair {
	t.Helper()
	bare := mkBareDir(t)
	originURL := seedBareOrigin(t, bare)

	subBare := mkBareDir(t)
	subOrigin := seedBareOrigin(t, subBare)

	workspace := mkProjectDir(t)
	cloneFromOrigin(t, workspace, originURL)
	gitRunC(t, workspace, "submodule", "add", subOrigin, "submod")
	gitRunC(t, workspace, "commit", "-m", "Add submodule")
	gitRunC(t, workspace, "push", "origin", "main")

	remoteDir := mkProjectDir(t)
	localDir := mkLocalDir(t)
	cloneFromOrigin(t, remoteDir, originURL)
	cloneFromOrigin(t, localDir, originURL)
	gitRunC(t, remoteDir, "submodule", "update", "--init", "--recursive")
	gitRunC(t, localDir, "submodule", "update", "--init", "--recursive")
	return RepoPair{RemoteDir: remoteDir, LocalDir: localDir}
}

func dirtySubmoduleFile(t *testing.T, parent string) string {
	t.Helper()
	p := filepath.Join(parent, "submod", "README.md")
	if err := os.WriteFile(p, []byte("dirty inside submodule\n"), 0644); err != nil {
		t.Fatalf("dirty submodule: %v", err)
	}
	return "submod"
}
```