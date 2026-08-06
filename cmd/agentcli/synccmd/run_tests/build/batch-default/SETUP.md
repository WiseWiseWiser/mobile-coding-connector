# Scenario

**Feature**: BuildUnisonCmd default batch argv includes profile + hostname env

```
seed mad-max (batch=true)
  -> BuildUnisonCmd(Name=mad-max)
  -> argv has remote-agent-mad-max + -batch; env UNISONLOCALHOSTNAME
```

## Preconditions

- Pair mad-max with LocalHostname `remote-agent-mad-max`, Batch true.
- Profile seeded (builder may not require file, but store pair exists).
- Interactive false (default).

## Steps

1. Seed mad-max pair + profile.
2. Mode build; optional LocalUnisonPath for binary identity.
3. Assert profile token, batch flag, and hostname env.

## Context

- Core argv/env contract for non-interactive run.

```go
import (
	"path/filepath"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Mode = "build"
	seedMadMax(req)
	req.Interactive = false
	// Distinct binary so argv[0] is unambiguous.
	req.LocalUnisonPath = filepath.Join(req.UnisonDir, "fake-unison-bin")
	return nil
}
```
