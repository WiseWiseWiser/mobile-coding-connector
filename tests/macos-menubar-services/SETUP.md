# Scenario

**Feature**: macOS menu bar Services submenu formatting and Swift contract

```
ServiceStatus -> FormatServiceTitle / action gating / alert copy -> strings
Swift sources -> ServerClient on :23712, nested Menu, /api/logs/stream SSE
```

## Preconditions

1. `macosapp/menubar` exports `FormatServiceTitle`, `CanStopService`,
   `ShowEnableAction`, `DisableAlertMessage`, `EnableAlertMessage`, and
   `FormatServicesEmptyLabel`.
2. Go formatter leaves are pure function calls — no network or subprocess.
3. Client leaves read Swift sources under `macos-ai-critic/ai-critic-macos/`.

## Steps

1. Leaf `Setup` sets `Op` and formatter-specific inputs (or `ClientLeaf` for Swift).
2. Root `Run` dispatches by `Op` to formatters or source inspection.
3. Leaf `Assert` checks exact strings, booleans, or Swift contract fields.

## Context

Implements REQUIREMENT-DESIGN-menubar-services-server-port.md. Primary logic lives in
Go (`macosapp/menubar/`); Swift mirrors the same contracts.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	return nil
}
```