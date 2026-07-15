# Scenario

**Feature**: auth failure on paste-bin

```
# bad token -> GET/PUT scratch rejected -> non-zero exit
bad --token -> unauthorized error
```

## Preconditions

Scratch seeded; token intentionally invalid.

## Steps

1. `seedScratch(req, seededUTF8Content, "")`.
2. `setReadTTY(req)`.
3. `req.Token = "definitely-not-the-test-password"`.

## Context

REQUIREMENT leaf: `auth-failure`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	seedScratch(req, seededUTF8Content, "")
	setReadTTY(t, req)
	req.Token = "definitely-not-the-test-password"
	return nil
}
```