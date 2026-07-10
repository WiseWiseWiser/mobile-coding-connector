# Scenario

**Feature**: macOS menu bar Cron formatting, cronapi request builders, and Swift contracts

```
# pure helpers
CronTaskStatus(name,status,enabled,schedule) -> FormatCronTaskTitle / CanRun / ShowEnable / empty labels
server message or empty -> CronToggleAlertMessage -> NSAlert copy

# request builders (mirror serviceapi)
baseURL + token + action/id -> cronapi Build*Request -> Method/URL/Authorization

# Swift apps
local AICriticApp  -> Menu("Cron") after Services before Terminals; ServerClient; SSE logs
remote AICriticApp -> same Cron UX; ServiceClient base URL + Bearer; Not configured path
30s refresh + top-level Refresh -> list cron tasks with services/terminals
```

## Preconditions

1. `macosapp/menubar` exports `FormatCronTaskTitle`, `CanRunCronTask`,
   `ShowEnableCronAction`, `FormatCronTasksEmptyLabel`,
   `FormatCronNotConfiguredLabel`, and `CronToggleAlertMessage`.
2. `macosapp/cronapi` exports list/action path helpers and
   `BuildListCronTasksRequest` / `BuildCronActionRequest` with optional Bearer auth.
3. Go helper leaves are pure function calls — no network or subprocess.
4. Client leaves read Swift sources under `macos-ai-critic/ai-critic-macos/` and
   `macos-ai-critic/ai-critic-remote-macos/` (plus Shared/).

## Steps

1. Leaf `Setup` sets `Op` and inputs (or `ClientLeaf` / `CronAPILeaf`).
2. Root `Run` dispatches by `Op` to helpers, cronapi builders, or source inspection.
3. Leaf `Assert` checks exact strings, booleans, paths, or Swift contract flags.

## Context

Implements REQUIREMENT-DESIGN-macos-menubar-cron.md. Primary logic lives in Go
(`macosapp/menubar`, `macosapp/cronapi`); Swift mirrors for UI. RED until
formatters, cronapi package, and Cron menus land.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	// Root: no shared mutation; leaves set Op and case inputs.
	_ = t
	_ = req
	return nil
}
```
