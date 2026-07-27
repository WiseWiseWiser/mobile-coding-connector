# Scenario

**Feature**: API validation and not-found errors

```
# unknown id -> 404; empty name -> 400
```

## Preconditions

1. Mode api (parent); error-focused leaves.

## Steps

1. Mark Token present for Bearer wrapper.
2. Leaf issues bad request.
3. Assert 4xx class.

## Context

Error contract.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	// Errors group always exercises authenticated HTTP paths.
	if req.Token == "" {
		req.Token = "test-token"
	}
	req.Mode = "api"
	return nil
}
```
