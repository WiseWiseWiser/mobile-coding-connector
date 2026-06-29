# Scenario

**Feature**: extension startup is slow but must not block daemon readiness

```
# opencode.json triggers extension path; hook sleeps at RunExtensionStartup entry
test env (delay ms) -> server extension_start (delayed) -> keep-alive still sees /ping <10s
```

## Preconditions

- `WriteExtensionConfig` is true for descendants unless overridden.
- `AI_CRITIC_TEST_EXTENSION_DELAY_MS` is set by leaf `Setup` (default 15000 ms).

## Steps

1. Write minimal `opencode.json` with external domain and `WebServer.Enabled=true`.
2. Ensure `SkipExtensionStartup` is false.

## Context

Uses test hooks only — no live Cloudflare tunnel provisioning in doctests.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.WriteExtensionConfig = true
	req.SkipExtensionStartup = false
	if req.ExtensionDelayMs <= 0 {
		req.ExtensionDelayMs = 15000
	}
	return nil
}
```