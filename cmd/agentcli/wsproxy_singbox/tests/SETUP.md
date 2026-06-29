# Scenario

**Feature**: ws-proxy sing-box doctest harness with injectable hooks

```
# leaf Setup narrows Request; root Run installs hooks and dispatches Op
doctest Setup chain -> InstallTestHooks -> singbox.Run* -> Response + audit fields

# no real brew, sudo, or sing-box subprocesses
Hook layer -> mock LookPath/Confirm/BrewInstall/RunSingBox/StartDetached/FetchVMess
```

## Preconditions

- Target package `cmd/agentcli/wsproxy_singbox` exposes `InstallTestHooks`,
  `RunClientConfig`, `RunTun`, and `BuildSingBoxTunConfig`.
- Hooks replace all third-party process invocation; doctests never call real
  `brew`, `sudo`, or `sing-box`.
- `UserCacheDir` hook redirects cache to `t.TempDir()` for isolation.

## Steps

1. Ancestor `Setup` sets `Request.Op` and scenario-specific hook overrides.
2. Root `Run` installs hooks, captures stdout/stderr, executes operation.
3. Leaf `Assert` validates `Response` and hook audit trail.

## Context

- Default mock VMess uses `ws-test.example.com:443` path `/ws` with TLS enabled.
- Default non-root EUID is `1000` when `Request.EUID` is nil; set `Request.EUID` to pointer `0` for root scenarios.
- Detach PID defaults to `4242` when `Request.DetachPID` is zero.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.MockVMess == nil && req.Op != OpBuildConfig && req.Op != OpBuildHttpOnlyConfig && req.Op != OpParsePolicy {
		req.MockVMess = defaultMockVMess()
	}
	return nil
}

func euidPtr(v int) *int {
	return &v
}
```