# Scenario

**Feature**: remote macOS menu bar config, profile, and source contracts

```
# pure Go mirrors for Swift remote product
remote-agent-config.json -> macosapp/remoteconfig.Resolve/Save/FormatStatus
token -> AuthorizationHeader -> "Bearer …"
profile remote|local -> appprofile flags (SpawnsDaemon, config file, display name)
install-remote.sh / Swift sources -> identity + no Restart Daemon (remote)
```

## Preconditions

1. Implementer provides pure packages:
   - `macosapp/remoteconfig` — `Config`, `Domain`, `ResolvedEndpoint`,
     `ConnectionState`, `Load`, `Save`, `Resolve`, `AuthorizationHeader`,
     `FormatStatus`, `OpenBrowserURL` (and server normalize rules).
   - `macosapp/appprofile` — `Local()` / `Remote()` profile flags.
2. No network for resolve/save/auth/status/profile/browser leaves (temp dirs for I/O).
3. Client leaves are read-only source inspection (may be RED until product files exist).

## Steps

1. Leaf `Setup` sets `Op` and scenario inputs.
2. Root `Run` dispatches by `Op`.
3. Leaf `Assert` checks state, strings, file mode, or source contract fields.

## Context

REQUIREMENT-DESIGN-remote-agent-macos-bar-app.md. Local app behavior remains
unchanged; this tree locks the **remote** entry only.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	return nil
}
```
