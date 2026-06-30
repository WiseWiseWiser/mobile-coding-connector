# Auth Doctests

Package-level tests for `server/auth` verifying setup-flow endpoints on an
uninitialized server (no `server-credentials` file).

# DSN (Domain Specific Notion)

The auth doctest harness models first-launch setup: the server has no credentials
yet, the frontend shows the Setup page, and certain endpoints must work without
auth while others are incorrectly blocked.

**Participants**

- **Auth middleware** — wraps `/api/*`; returns `not_initialized` when credentials
  file is missing or empty.
- **Skip paths** — login, auth check/status/setup, ping, public key, path-info.
- **Setup page** — `POST /api/auth/credentials/generate` must return a 64-char hex
  credential before the server is initialized.
- **Setup endpoint** — `POST /api/auth/setup` writes the first credential (already
  in skip paths).

**Behaviors**

- Uninitialized server: `POST /api/auth/credentials/generate` returns 200 with
  `{"credential":"<64-char-hex>"}`.
- Uninitialized server: `POST /api/auth/setup` remains allowed (control).
- Current bug: generate is blocked by middleware with `{"error":"not_initialized"}`.

## Version

0.0.2

## Decision Tree

```
[auth setup on uninitialized server]
 |
 +-- setup/
      |
      +-- generate-credential-uninitialized/   (LEAF)
      |    POST /api/auth/credentials/generate → 200 + 64-char hex credential
      |
      +-- setup-endpoint-allowed/              (LEAF)
           POST /api/auth/setup → allowed through middleware (control)
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `setup/generate-credential-uninitialized` | Generate Random API works before credentials exist |
| 2 | `setup/setup-endpoint-allowed` | Setup endpoint is in skip paths (control) |

## How to Run

```sh
doctest vet ./server/auth/tests
doctest test ./server/auth/tests/...
doctest test ./server/auth/tests/setup/generate-credential-uninitialized
```

```go
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/xhd2015/ai-critic/server/auth"
)

const (
	OpGenerateCredential = "generate-credential"
	OpSetupEndpoint      = "setup-endpoint"
)

// Production skip paths from server/server.go Serve().
var authSkipPaths = []string{
	"/api/login",
	"/api/auth/check",
	"/api/auth/status",
	"/api/auth/setup",
	"/api/auth/credentials/generate",
	"/ping",
	"/api/encrypt/public-key",
	"/api/tools/path-info",
}

type Request struct {
	Op string
}

type Response struct {
	StatusCode int
	Body       string
	JSON       map[string]any
}

func prepUninitialized(t *testing.T) string {
	tmpDir := t.TempDir()
	credFile := filepath.Join(tmpDir, "server-credentials")
	auth.SetCredentialsFile(credFile)
	t.Cleanup(func() {
		auth.SetCredentialsFile("")
	})
	return credFile
}

func newAuthHandler() http.Handler {
	mux := http.NewServeMux()
	auth.RegisterAPI(mux)
	return auth.Middleware(mux, authSkipPaths)
}

func doRequest(t *testing.T, handler http.Handler, method, path string, body []byte) *Response {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	req := httptest.NewRequest(method, path, bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	resp := &Response{
		StatusCode: rr.Code,
		Body:       rr.Body.String(),
	}
	if resp.Body != "" {
		var parsed map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &parsed); err == nil {
			resp.JSON = parsed
		}
	}
	return resp
}

func Run(t *testing.T, req *Request) (*Response, error) {
	if req.Op == "" {
		return nil, fmt.Errorf("Op is required")
	}

	prepUninitialized(t)
	handler := newAuthHandler()

	switch req.Op {
	case OpGenerateCredential:
		return doRequest(t, handler, http.MethodPost, "/api/auth/credentials/generate", nil), nil
	case OpSetupEndpoint:
		payload := []byte(`{"credential":"abc123def456"}`)
		return doRequest(t, handler, http.MethodPost, "/api/auth/setup", payload), nil
	default:
		return nil, fmt.Errorf("unknown Op: %s", req.Op)
	}
}
```