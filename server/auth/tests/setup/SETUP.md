# Scenario

**Feature**: auth setup endpoints on uninitialized server

```
# no server-credentials file
Uninitialized server -> Auth middleware -> setup-flow endpoints
```

## Preconditions

- Credentials file path points to a non-existent file under a temp directory.

## Steps

1. Child leaf sets `Request.Op` for generate or setup control.

## Context

Grouping for first-launch setup API behavior.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.Op == "" {
		req.Op = OpGenerateCredential
	}
	return nil
}
```