# Scenario

**Feature**: backup dry-run excludes executable binaries via **(binary) rule

```
# IsExecutableBinary on ELF stub -> EXCLUDED lists **(binary); stub omitted from DOT FILES
```

## Preconditions

`serverHome` includes ELF stub at `.ai-critic/bin/stub`.

## Steps

1. Args: `machine backup --dry-run`.

## Context

REQUIREMENT leaf `backup/binary-exclude`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Args = []string{"machine", "backup", "--dry-run"}
	return nil
}
```