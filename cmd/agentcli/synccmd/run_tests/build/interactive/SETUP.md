# Scenario

**Feature**: BuildUnisonCmd interactive omits -batch

```
seed mad-max -> BuildUnisonCmd(Interactive=true)
  -> argv has profile; no -batch token
```

## Preconditions

- Same pair as batch-default; Interactive true.

## Steps

1. Seed mad-max.
2. Set Interactive true.
3. Assert no `-batch` in argv; hostname env still set.

## Context

- Flag interaction: interactive overrides pair.Batch for argv.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Mode = "build"
	seedMadMax(req)
	req.Interactive = true
	return nil
}
```
