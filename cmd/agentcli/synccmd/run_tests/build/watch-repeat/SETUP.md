# Scenario

**Feature**: BuildUnisonCmd with Watch appends -repeat watch

```
seed mad-max (batch=true)
  -> BuildUnisonCmd(Watch=true)
  -> argv has -batch and adjacent -repeat watch
```

## Preconditions

- Pair mad-max with Batch true.
- Interactive false; Watch true.

## Steps

1. Seed mad-max pair + profile.
2. Mode build; Watch true; LocalUnisonPath set.
3. Assert profile, batch, and `-repeat watch`.

## Context

- Continuous-mode argv contract for `--watch`.

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
	req.Watch = true
	req.LocalUnisonPath = filepath.Join(req.UnisonDir, "fake-unison-bin")
	return nil
}
```
