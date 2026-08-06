# Scenario

**Feature**: Doctor success path — all checks pass

```
Doctor(name, happy fakes) -> AllOK true, nil/empty DoctorErr
```

## Preconditions

- Grouping for successful doctor outcomes.
- Happy probe hooks + existing local root + profile.

## Steps

1. Leaves seed one pair, profile, and happy hooks.
2. Assert every stable check name OK.

## Context

- Primary greenpath leaf under `all-ok/`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Mode = "doctor"
	return nil
}
```
