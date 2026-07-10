# Scenario

**Feature**: cron scheduler runtime rules (interval, overlap, timeout, enable, run, cron UTC)

```
# manager tick loop evaluates due tasks
create/seed task -> tick -> bash -lc command -> log + history + status
# locked: finish+interval, skip overlap, timeout always, UTC cron
```

## Preconditions

1. Short intervals and marker files keep runtime leaves under ~30s.
2. Long-running commands use `sleep` for overlap/timeout evidence.

## Steps

1. Leaf configures schedule, command, timeout, enabled, and wait/poll.
2. Run creates or seeds, waits for scheduler evidence.
3. Assert checks run counts, PIDs, nextRunAt, errors, stored cron expr.

## Context

Priority leaves 2–7, 11 (default timeout). Unit-style pure schedule math may
also live in package tests during implementation; these leaves are the sealed E2E contract.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.UseCLI = false
	if req.PollTimeoutSecs <= 0 {
		req.PollTimeoutSecs = 25
	}
	return nil
}
```
