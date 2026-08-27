# Scenario

**Feature**: BuildUnisonCmd with IntervalSeconds appends -repeat N

```
seed mad-max (batch=true)
  -> BuildUnisonCmd(IntervalSeconds=60)
  -> argv has -batch and adjacent -repeat 60
```

## Preconditions

- Pair mad-max with Batch true.
- Interactive false; IntervalSeconds 60; Watch false.

## Steps

1. Seed mad-max pair + profile.
2. Mode build; IntervalSeconds 60.
3. Assert profile, batch, and `-repeat 60`.

## Context

- Polled continuous-mode argv for `--interval`.

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
	req.Watch = false
	req.IntervalSeconds = 60
	req.LocalUnisonPath = filepath.Join(req.UnisonDir, "fake-unison-bin")
	return nil
}
```
