# Scenario

**Feature**: Setup page Generate Random on uninitialized server

```
Setup page -> click Generate Random -> credential fills input (no error)
```

## Preconditions

1. Server has no credentials (first launch).
2. Browser shows Setup page with "Generate Random" button.

## Steps

1. Set `Request.Uninitialized = true`.
2. Set `Request.ScriptPath = "script.js"`.
3. Playwright navigates to `BASE_URL`, clicks Generate Random, reads input and error.

## Context

End-to-end reproduction of the user report: clicking Generate Random shows
`not_initialized` instead of filling the credential field.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Uninitialized = true
	req.ScriptPath = "script.js"
	if req.TimeoutSecs <= 0 {
		req.TimeoutSecs = 120
	}
	return nil
}
```