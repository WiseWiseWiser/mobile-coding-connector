# Scenario

**Feature**: local menu-bar resolves Bearer token for loopback ServerClient

```
# locked order: config → credentials → none (fall-through on read/empty only)
{DataDir}/local-agent-config.json -> localauth.ResolveLocalServerToken -> token + source=config
{DataDir}/server-credentials     -> first non-empty line              -> source=credentials
(no usable token)                -> token ""                          -> source=none

# ServerClient request path
token -> AuthorizationHeader / Swift ServerClient -> Authorization: Bearer <token>
```

## Preconditions

1. Implementer provides pure package `macosapp/localauth`:
   - `ResolveLocalServerToken(Options) (token string, source TokenSource)`
   - `TokenSource` values: `config` | `credentials` | `none`
   - `Options.DataDir` — empty defaults to `~/.ai-critic`; tests always pass temp dirs
   - `AuthorizationHeader(token string) string` — `Bearer …` or empty
2. Resolve leaves use temp `DataDir` only (never real home).
3. Client leaves are read-only Swift source inspection (RED until ServerClient auth wiring exists).
4. No network required.

## Steps

1. Leaf `Setup` sets `Op` and fixture fields (`ConfigPresent`, `CredentialsPresent`, …).
2. Root `Run` dispatches by `Op`; for `resolve`, materializes files under temp `DataDir`.
3. Leaf `Assert` checks `Token`/`Source`, header string, profile flag, or source-contract flags.

## Context

REQUIREMENT-DESIGN-local-menubar-token-resolve.md. Remote menu-bar
(`remote-agent-config.json`) is out of scope.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	// Root: no shared fixture mutation; leaves/groups fill Request.
	// Ensure zero-value Request is a clean slate for each leaf package.
	*req = Request{}
	return nil
}
```
