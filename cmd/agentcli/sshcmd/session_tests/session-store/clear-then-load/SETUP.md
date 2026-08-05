# Scenario

**Feature**: Clear after Save makes subsequent Load return nil

```
# clear removes profile session
tests -> Save(sess) -> Clear(profileID) -> Load -> (nil, nil)
```

## Preconditions

- Scenario: `session-clear`.
- SessionToSave: Alive sample session for the profile.

## Steps

1. Save sample session.
2. Clear profile.
3. Load; Assert nil session and no error.

## Context

- Serve teardown uses Clear so P1 client gate fails Alive.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Scenario = ScenarioSessionClear
	req.SessionToSave = sampleSession(req.ProfileID, req.ConfigDir, 18022, 0, true)
	return nil
}
```
