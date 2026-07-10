# Scenario

**Feature**: macOS menu bar Cron create / update / delete (CRUD) + Cron Editor

```
# request builders
CronTaskDef + baseURL/token -> cronapi BuildCreate|Update|Delete -> Method/URL/Body/Authorization

# delete gating + confirm
status -> CanDeleteCronTask -> bool (false only when running)
name -> FormatDeleteCronConfirm -> `Delete cron task "{name}"?`

# local↔UTC (align CLI)
local cron expr + fixed-offset loc -> ConvertLocalCronToUTC -> UTC expr | error
UTC expr + loc -> ConvertUTCCronToLocal -> local expr | error (edit open)

# Swift apps
local AICriticApp  -> Menu("Cron"): Edit…/Delete… + New Cron Task…; ServerClient CRUD
remote AICriticApp -> same; ServiceClient base+Bearer; New disabled if not configured
CronEditorView Save -> create (POST) or update (PUT) -> refresh -> close
```

## Preconditions

1. `macosapp/cronapi` exports create/update/delete path helpers,
   `BuildCreateCronTaskRequest` / `BuildUpdateCronTaskRequest` /
   `BuildDeleteCronTaskRequest`, `CronTaskDef`, optional `Body` on
   `CronRequest`, and `ConvertLocalCronToUTC` / `ConvertUTCCronToLocal`.
2. `macosapp/menubar` exports `CanDeleteCronTask` and
   `FormatDeleteCronConfirm`.
3. Go helper leaves are pure function calls — no network or subprocess.
4. Client leaves read Swift sources under `macos-ai-critic/ai-critic-macos/`,
   `macos-ai-critic/ai-critic-remote-macos/`, and `macos-ai-critic/Shared/`.

## Steps

1. Leaf `Setup` sets `Op` and inputs (`CronAPILeaf` / `ConvertLeaf` /
   `ClientLeaf` / status / body fields).
2. Root `Run` dispatches by `Op` to builders, formatters, convert, or source
   inspection.
3. Leaf `Assert` checks methods, paths, body fields, booleans, convert
   results, or Swift contract flags.

## Context

Implements REQUIREMENT-DESIGN-macos-menubar-cron-crud.md. Primary logic lives
in Go (`macosapp/cronapi`, `macosapp/menubar`); Swift mirrors for UI. RED until
CRUD builders, convert helpers, delete gating, and Cron Editor menus land.
Does not modify operate-only tree `tests/macos-menubar-cron/`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	// Root: no shared mutation; leaves set Op and case inputs.
	_ = t
	_ = req
	return nil
}
```
