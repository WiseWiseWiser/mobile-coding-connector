# Scenario

**Feature**: RunPair non-zero exit writes state and surfaces error

```
seed mad-max + FakeExitCode 3
  -> RunPair -> Exec called; state exitCode 3; RunPairErr non-empty
```

## Preconditions

- Happy doctor; FakeExitCode 3.

## Steps

1. Seed mad-max + happy hooks.
2. FakeExitCode 3.
3. Assert error mentions exit / 3; state exitCode 3.

## Context

- Map Unison non-zero exit to library error while persisting state.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Mode = "run"
	seedMadMax(req)
	req.FakeExitCode = 3
	return nil
}
```
