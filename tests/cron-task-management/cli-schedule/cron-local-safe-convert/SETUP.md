# Scenario

**Feature**: CLI `--cron` converts safe local expression to UTC and prints both

```
# TZ=Etc/GMT-8 (fixed UTC+8, no DST), --cron "0 9 * * *"
CLI -> local 09:00 -> stored UTC "0 1 * * *"; stdout shows local + stored UTC
```

## Preconditions

1. Fixed-offset timezone without DST: `Etc/GMT-8` means UTC+8.
2. Simple expr `M H * * *` is considered **safe** to convert.
3. Expected UTC hour: 9 - 8 = 1 → `0 1 * * *`.

## Steps

1. `remote-agent cron add --name … --command … --cron "0 9 * * *"` with `TZ=Etc/GMT-8`.
2. Assert exit 0, stdout mentions both local and UTC forms, list stores UTC expr.

## Context

Priority leaf: CLI `--cron` local convert. On success print both local and stored UTC.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.UseCLI = true
	req.Action = "create"
	req.TaskName = "local-morning"
	req.CLIEnv = []string{"TZ=Etc/GMT-8"}
	req.CLIArgs = []string{
		"cron", "add",
		"--name", "local-morning",
		"--command", "echo local-cron",
		"--cron", "0 9 * * *",
		"--timeout", "1h",
	}
	return nil
}
```
