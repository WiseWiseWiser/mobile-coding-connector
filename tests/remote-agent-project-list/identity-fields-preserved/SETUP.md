# Scenario

**Feature**: git identity fields remain alongside new git status lines

```
# clean repo + saved git identity on project record
remote-agent project list -> git status lines + Git Identity ID/Name/Email unchanged
```

## Preconditions

Git repo with initial commit; project row includes git identity metadata.

## Steps

1. Create clean repo on `main` with commit `Initial commit`.
2. Register project `identity-test` (`identity-001`) with:
   - `git_user_config_id`: `mp663i1zlyx3th`
   - `git_user_name`: `xhd2015`
   - `git_user_email`: `xhd2015@gmail.com`

## Context

REQUIREMENT leaf: `identity-fields-preserved/` — identity lines still present plus
new git status lines inserted after `Dir` and before identity fields.

```go
import (
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	dir := mkProjectDir(t)
	gitInitWithMain(t, dir)
	gitInitialCommit(t, dir, "Initial commit")

	req.Project = ProjectEntry{
		ID:              "identity-001",
		Name:            "identity-test",
		Dir:             dir,
		GitUserConfigID: "mp663i1zlyx3th",
		GitUserName:     "xhd2015",
		GitUserEmail:    "xhd2015@gmail.com",
	}
	return nil
}
```