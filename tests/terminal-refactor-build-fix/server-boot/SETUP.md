# Scenario

**Feature**: ai-critic server boots and registers routes without panic

```
# server boot route registration (mirrors `ai-critic-server keep-alive` spawn)
server.Serve -> server.RegisterAPI(mux) -> terminal.RegisterAPI(mux) -> ptywrap routes
```

## Preconditions

- `github.com/xhd2015/ai-critic/server/terminal` exports `RegisterAPI(mux)`.
- A fresh `net/http.ServeMux` (Go 1.22+ pattern mux) panics on duplicate or
  conflicting pattern registration.

## Steps

1. Leaf sets `req.Phase` to a server-boot smoke scenario.
2. Harness calls the relevant `RegisterAPI` on a fresh mux inside a `recover`,
   capturing any panic as response fields so the test process is not killed.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.Phase == "" {
		req.Phase = "server-api-register"
	}
	return nil
}
```
