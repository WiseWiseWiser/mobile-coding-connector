# Scenario

**Feature**: generate credential on uninitialized server

```
POST /api/auth/credentials/generate -> 200 + {"credential":"<64-char-hex>"}
```

## Preconditions

- No `server-credentials` file exists.
- Server is in uninitialized state (same as first launch).

## Steps

1. Set `Request.Op = OpGenerateCredential`.
2. Issue `POST /api/auth/credentials/generate` through auth middleware.

## Context

This is the API behind the Setup page "Generate Random" button. README documents
that users can click Generate Random before confirming setup. The endpoint must
not return `not_initialized`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpGenerateCredential
	return nil
}
```