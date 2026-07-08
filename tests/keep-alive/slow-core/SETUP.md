# Scenario

**Bug**: keep-alive kills server when core bind is slow under I/O pressure

```
# test hook sleeps before net.Listen; daemon must honor --startup-timeout
test env (core delay ms) -> server Serve (delayed listen) -> keep-alive WaitForPort
```

## Preconditions

- `AI_CRITIC_TEST_CORE_DELAY_MS` is set by descendant `Setup` (default 15000 ms).
- `SkipExtensionStartup` is true — extension path must not mask core bind delay.
- No `opencode.json` extension config unless a leaf overrides.

## Steps

1. Set `AI_CRITIC_TEST_SKIP_EXTENSION=1` on the managed server child.
2. Descendant leaves set `StartupTimeout` and `ObserveSecs`.

## Context

Complements `slow-extension/` (delay after core bind). Models remote PH host
fork/exec + I/O pressure where port never opens within 10s.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.WriteExtensionConfig = false
	req.SkipExtensionStartup = true
	req.ExtensionDelayMs = 0
	if req.CoreDelayMs <= 0 {
		req.CoreDelayMs = 15000
	}
	return nil
}
```