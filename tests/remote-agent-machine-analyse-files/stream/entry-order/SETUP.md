# Scenario

**Feature**: entry blocks stream in alphabetical order by entry name

```
# multiple top-level entries with sortable names
blocks appear as .codex, aaa-first, mmm-mid, notes.txt, zzz-last
```

## Preconditions

`SeedProfile=entry-order`: `.codex`, `aaa-first`, `mmm-mid`, `notes.txt`, `zzz-last`.

## Steps

1. Set `SeedProfile` to `entry-order`.

## Context

REQUIREMENT leaf `stream/entry-order`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SeedProfile = "entry-order"
	req.Args = []string{"machine", "analyse-files"}
	return nil
}
```