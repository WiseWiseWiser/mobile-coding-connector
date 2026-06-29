# Scenario

**Feature**: core-only bootstrap without extension work

```
# skip extension hook; no opencode auto-start config
keep-alive -> server core_ready (fast) -> /ping; no tunnel/extension task logs
```

## Preconditions

- Extension skipped via `AI_CRITIC_TEST_SKIP_EXTENSION=1`.
- No `opencode.json` extension trigger (optional disabled stub).

## Steps

1. Set `SkipExtensionStartup=true`, `ExtensionDelayMs=0`.
2. Write disabled `opencode.json` so auto-start does not arm tunnels.

## Context

Baseline for fast startup when extension path is inactive.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SkipExtensionStartup = true
	req.ExtensionDelayMs = 0
	req.WriteExtensionConfig = false
	if req.ObserveSecs <= 0 {
		req.ObserveSecs = 8
	}
	return nil
}
```