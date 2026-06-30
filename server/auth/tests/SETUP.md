# Scenario

**Bug**: Generate Random blocked on uninitialized server

```
# no server-credentials file
Uninitialized server -> Auth middleware -> POST /api/auth/credentials/generate
```

## Preconditions

Tests run against the auth middleware stack with an isolated temporary config
home that has **no** `server-credentials` file. The harness mirrors production
`server.Serve` skip paths.

## Steps

1. Child `Setup` sets `Request.Op` for the scenario.
2. Root `Run` points credentials file at a non-existent path under temp dir.
3. `Run` registers auth API on a mux, wraps with `auth.Middleware`, issues HTTP.
4. Leaf `Assert` validates the response.

## Context

Reproduces the Setup page "Generate Random" bug: clicking the button calls
`POST /api/auth/credentials/generate`, which the auth middleware blocks with
`not_initialized` even though setup endpoints should work before initialization.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.Op == "" {
		req.Op = OpGenerateCredential
	}
	return nil
}
```