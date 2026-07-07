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
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SeedProfile = "basic"
	req.Args = []string{"machine", "analyse-files"}
	return nil
}
```