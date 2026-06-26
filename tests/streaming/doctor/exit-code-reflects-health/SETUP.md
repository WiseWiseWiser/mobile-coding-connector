# Scenario

**Feature**: doctor exits non-zero when checks fail

```
# fixture lacks tunnel mapping -> unhealthy -> exit 1
Result: unhealthy -> non-zero exit code
```

## Preconditions

Standard fixture (tunnel mapping absent) produces failing checks.

## Steps

No additional delay — default harness.

## Context

Extended integration coverage for CLI exit semantics.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.RecordLineTimes = false
	return nil
}
```
