# Scenario

**Feature**: setup endpoint allowed on uninitialized server (control)

```
POST /api/auth/setup -> reaches handler (not blocked by not_initialized)
```

## Preconditions

- No `server-credentials` file exists.
- `/api/auth/setup` is in auth middleware skip paths.

## Steps

1. Set `Request.Op = OpSetupEndpoint`.
2. Issue `POST /api/auth/setup` with a sample credential body.

## Context

Control test: setup is already in skip paths and must not return `not_initialized`.
Generate should behave the same way for first-launch UX.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpSetupEndpoint
	return nil
}
```