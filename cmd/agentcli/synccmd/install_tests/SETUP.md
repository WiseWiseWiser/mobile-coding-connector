# Scenario

**Feature**: remote-agent sync unison P4 — install (LocalEnsure / RemoteEnsure + CLI)

```
# library path
operator -> synccmd.Install(opts{Scope, LocalEnsure, RemoteEnsure})
  -> InstallReport{LocalVersion, RemoteVersion, LocalOK, RemoteOK, Messages}

# CLI path
operator -> synccmd.RunCLI([unison install …], CLIOpts{hooks})
  -> stdout/stderr; exit via RunCLI error
```

## Preconditions

- Package: `github.com/xhd2015/ai-critic/cmd/agentcli/synccmd` (P1–P3 GREEN; P4
  APIs may be missing → suite **RED** on compile/link until implementer).
- Public surface under test: `Install`, `InstallOpts`, `InstallReport`,
  `PreferredUnisonVersion`, extended `CLIOpts` / `RunCLI` / `UnisonUsage`.
- Injectable hooks only — no brew, HTTP download, process env/cwd mutation.
- Parallel-safe: paths under `d.DOCTEST_CASE` when RemoteTargetPath is set.
- No real Unison binary, SSH, or network.

## Steps

1. Root Setup assigns default `RemoteTargetPath` under `d.DOCTEST_CASE` when empty.
2. Grouping/leaf Setup set Mode, Scope, Args, and Fake* / hook error fields.
3. Root `Run` wraps hooks for call capture then dispatches Install or RunCLI.

## Context

- Module: `github.com/xhd2015/ai-critic`
- Tree home: `cmd/agentcli/synccmd/install_tests`
- Spec version 0.0.2; L2 in-process only.
- P1 `sync_tests/`, P2 `doctor_status_tests/`, P3 `run_tests/` remain sealed and GREEN.

```go
import (
	"path/filepath"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	if req.RemoteTargetPath == "" {
		req.RemoteTargetPath = filepath.Join(d.DOCTEST_CASE, "remote-bin", "unison")
	}
	return nil
}
```
