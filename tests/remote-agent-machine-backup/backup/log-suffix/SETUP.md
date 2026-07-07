# Scenario

**Feature**: backup dry-run excludes **/*.log files by basename suffix

```
# walk applies .log suffix rule -> service.log in EXCLUDED
```

## Preconditions

`serverHome` includes `.ai-critic/service.log` (excluded) and `.ai-critic/config.json` (included).

## Steps

1. Args: `machine backup --dry-run`.

## Context

REQUIREMENT leaf `backup/log-suffix`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Args = []string{"machine", "backup", "--dry-run"}
	return nil
}
```