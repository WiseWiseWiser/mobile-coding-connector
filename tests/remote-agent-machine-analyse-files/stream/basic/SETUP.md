# Scenario

**Feature**: analyse-files streams home line, entry blocks, and summary

```
# scan every serverHome child -> stream blocks -> summary rollups
remote-agent machine analyse-files -> home: + > headers + analyse-files summary
```

## Preconditions

`SeedProfile=basic`: `plain-dir/sub/nested.txt` and `notes.txt`.

## Steps

1. Set `SeedProfile` to `basic`.
2. Args: `machine analyse-files`.

## Context

REQUIREMENT leaf `stream/basic`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	// L3 smoke: analyse-files happy path via product binaries.
	req.UseCLI = true
	req.SeedProfile = "basic"
	req.Args = []string{"machine", "analyse-files"}
	return nil
}
```