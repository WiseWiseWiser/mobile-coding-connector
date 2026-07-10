# Scenario

**Feature**: remote Swift contracts for progress window + one-shot wiring

```
# ai-critic-remote-macos + Shared (read-only)
AICriticApp -> Backup Now… / enable-immediate -> BackupProgressWindow
MachineBackupClient -> yield SSE progress (not token-only)
hourly tick (triggeredBySchedule=true) -> no window
Backup Now .disabled: endpoint | running only (not backupEnabled)
```

## Preconditions

Swift sources under `macos-ai-critic/ai-critic-remote-macos/` and Shared.
Pure source inspection — no subprocess, UI, or network.

## Steps

1. Set `Op=client` and leaf-specific `ClientLeaf`.

## Context

REQUIREMENT Swift scenarios 16–20; enable-immediate window (goal 3).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "client"
	return nil
}
```
