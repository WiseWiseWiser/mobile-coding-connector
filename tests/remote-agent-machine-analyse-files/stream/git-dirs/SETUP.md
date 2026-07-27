# Scenario

**Feature**: git repo discovery adds git-dirs aggregate per entry

```
# with-git has git init; plain-dir has none
with-git block shows git-dirs 1; plain-dir omits git-dirs line
```

## Preconditions

`SeedProfile=git-dirs`: `with-git/` is a git repo; `plain-dir/` is not.

## Steps

1. Set `SeedProfile` to `git-dirs`.

## Context

REQUIREMENT leaf `stream/git-dirs`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.SeedProfile = "git-dirs"
	req.Args = []string{"machine", "analyse-files"}
	return nil
}
```