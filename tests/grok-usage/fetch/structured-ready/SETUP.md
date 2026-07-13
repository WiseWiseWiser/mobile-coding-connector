# Scenario

**Feature**: mock bare-local success → structured reset_at / reset_display / time_left

```
# bare wall-clock next_reset (no TZ) → service derives absolute + UI fields
GROK_SHOW_USAGE_COMMAND=mock-success-no-tz.sh -> tty fetch -> GrokUsageResponse
  status=ready, next_reset raw, reset_at RFC3339, reset_display, time_left
```

## Preconditions

1. `mock-success-no-tz.sh` emits `Weekly limit: 61%` and `Next reset: July 17, 08:55`
   (no timezone suffix — bare local wall clock).
2. Service on success must set A+B structured fields (not only raw `next_reset`).

## Steps

1. Set `MockScript=mock-success-no-tz.sh`.

## Context

REQUIREMENT-DESIGN-usage-structured-reset-ab.md scenario 1 (Grok fetch success
structured fields). Classic TDD: RED until service populates `ResetAt`,
`ResetDisplay`, and `TimeLeft`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.MockScript = "mock-success-no-tz.sh"
	return nil
}
```
