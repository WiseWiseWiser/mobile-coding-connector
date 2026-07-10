# Scenario

**Feature**: local↔UTC cron expression conversion (align CLI)

```
# save path (local form → server UTC)
local 5-field + fixed-offset loc -> ConvertLocalCronToUTC -> UTC expr | error

# edit open (server UTC → local form)
UTC 5-field + loc -> ConvertUTCCronToLocal -> local expr | error
```

## Preconditions

`Op=convert` dispatches to `macosapp/cronapi` convert helpers. Semantics match
CLI `convertLocalCronToUTC` in `cmd/agentcli/cron.go`: simple tokens only;
fixed offset required; ranges/lists/steps and DST zones are unsafe.

## Steps

1. Leaf sets `ConvertLeaf`, expression, and `TZName` (e.g. `Etc/GMT-8`).

## Context

REQUIREMENT optional leaves: local→UTC safe/unsafe; UTC→local for edit open.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "convert"
	return nil
}
```
