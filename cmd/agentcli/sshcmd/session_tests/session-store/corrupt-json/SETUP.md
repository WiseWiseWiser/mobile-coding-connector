# Scenario

**Feature**: Load of corrupt session JSON returns an error

```
# bad JSON is an error (not silent nil)
tests -> write garbage to session path -> FileSessionStore.Load
  -> error (Loaded may be nil)
```

## Preconditions

- Scenario: `session-corrupt`.
- Harness writes non-JSON bytes to `{Root}/ssh-sessions/{profileID}.json`.

## Steps

1. Write corrupt file at the canonical session path.
2. Load; Assert LoadErr non-empty and Loaded nil.

## Context

- Distinguishes missing (nil,nil) from corrupt (error).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Scenario = ScenarioSessionCorrupt
	return nil
}
```
