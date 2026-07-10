# Scenario

**Feature**: valid Bearer reaches handler successfully

```
POST open Authorization: Bearer test-iterm2-token -> 200
```

## Preconditions

Credentials file contains `test-iterm2-token`.

## Steps

1. Set `BearerToken=test-iterm2-token`.

## Context

REQUIREMENT: Bearer required and accepted when valid.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.OmitAuth = false
	req.BearerToken = "test-iterm2-token"
	return nil
}
```
