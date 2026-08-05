# Scenario

**Feature**: Save then Load returns equal session fields

```
# round-trip
tests -> FileSessionStore.Save(sess) -> disk JSON
tests -> FileSessionStore.Load(profileID) -> Session equal fields
```

## Preconditions

- Scenario: `session-save-load`.
- SessionToSave: full sample with LocalPort, User, ConfigDir, ServePID=0, Alive=true.

## Steps

1. Build sample Session for profile `default` with known fields.
2. Run Save then Load; Assert field equality.

## Context

- ServePID 0 means no process liveness check on Load.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Scenario = ScenarioSessionSaveLoad
	req.SessionToSave = sampleSession(req.ProfileID, req.ConfigDir, 51234, 0, true)
	return nil
}
```
