# Scenario

**Feature**: create/update validation — schedule XOR and timeout > 0

```
# invalid body -> HTTP error (non-2xx) and no persisted task
POST /api/cron-tasks with both|neither schedule or timeout≤0 -> error
```

## Preconditions

1. Validation errors surface as non-success HTTP status with a clear body.
2. Successful create is not required; list should not gain a valid new task from bad input.

## Steps

1. Leaf sets `RawBody` (or fields that produce invalid schedule/timeout).
2. Run POSTs create; records `HTTPStatus` and `ActionError`.
3. Assert expects error path.

## Context

Priority leaf 12 (validation).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.UseCLI = false
	req.Action = "create"
	return nil
}
```
