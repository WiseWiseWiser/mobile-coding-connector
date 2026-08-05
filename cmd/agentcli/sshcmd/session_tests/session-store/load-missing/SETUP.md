# Scenario

**Feature**: Load with no session file returns nil session and nil error

```
# missing file is not an error
tests -> FileSessionStore.Load(profileID) -> (nil, nil)
```

## Preconditions

- Scenario: `session-load-missing`.
- No prior Save; Root may not contain ssh-sessions yet.

## Steps

1. Call Load for default profile without creating a file.
2. Assert Loaded nil and LoadErr empty.

## Context

- Matches P1 gate: missing session is "no active tunnel", not store I/O error.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Scenario = ScenarioSessionLoadMissing
	return nil
}
```
