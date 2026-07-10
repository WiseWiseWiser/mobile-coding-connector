# Scenario

**Feature**: POST handler opens dir via injectible OpenConfig

```
POST /api/local/iterm2/open {dir, mode, send}
  -> Handler.Open(dir, &Config{Mode, FollowUpCommands})
  -> 200 {} | 4xx/5xx {"error":...}
```

## Preconditions

1. Injected `Open` records dir/mode/send; optional real OpenConfig + fake osascript.
2. Leaves create temp dirs/files under `t.TempDir()`.

## Steps

1. Set `Op=open`.
2. Leaf configures body and injection flags.

## Context

REQUIREMENT scenarios 1–5, 7 (handler paths).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "open"
	return nil
}
```
