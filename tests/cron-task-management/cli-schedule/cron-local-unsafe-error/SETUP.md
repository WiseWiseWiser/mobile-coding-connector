# Scenario

**Feature**: CLI `--cron` rejects unsafe/complex local conversion

```
# complex local expr with hour range + DOW: "0 9-17 * * 1-5"
CLI --cron -> non-zero exit; message tells user to use --cron-utc
```

## Preconditions

1. Expression `0 9-17 * * 1-5` is **not safe** to auto-convert (ranges / DOW complexity).
2. TZ may be anything; conversion must still refuse.

## Steps

1. `cron add --cron "0 9-17 * * 1-5" …`
2. Assert non-zero exit and stderr/stdout mentions `--cron-utc`.

## Context

Priority leaf: unsafe convert → error. No task should be created (or list lacks name).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.UseCLI = true
	req.Action = "create"
	req.TaskName = "unsafe-cron"
	req.CLIEnv = []string{"TZ=Etc/GMT-8"}
	req.CLIArgs = []string{
		"cron", "add",
		"--name", "unsafe-cron",
		"--command", "echo no",
		"--cron", "0 9-17 * * 1-5",
		"--timeout", "1h",
	}
	return nil
}
```
