# Scenario

**Feature**: remote-agent sync unison P3 — run (BuildUnisonCmd + RunPair + CLI)

```
# library path
operator -> synccmd.BuildUnisonCmd|RunPair(opts{StoreDir, Exec, probes})
  -> argv/env | RunResult + state/<name>.json

# CLI path
operator -> synccmd.RunCLI([unison run ...], CLIOpts{Exec, probes})
  -> stdout/stderr writers; state after Exec
```

## Preconditions

- Package: `github.com/xhd2015/ai-critic/cmd/agentcli/synccmd` (P1+P2 GREEN; P3
  APIs may be missing → suite **RED** on compile/link until implementer).
- Public surface under test: `BuildUnisonCmd`, `RunPair`, `RunOpts`, `RunResult`,
  `ExecFunc`, extended `CLIOpts` / `RunCLI` / `UnisonUsage`.
- Injectable absolute dirs + Exec + doctor probe hooks only — no process env/cwd
  mutation.
- Parallel-safe: paths under `d.DOCTEST_CASE`.
- No real Unison binary, SSH, or network.

## Steps

1. Root Setup assigns absolute StoreDir / UnisonDir / SSHConfigDir under
   `d.DOCTEST_CASE` when empty; creates those directories and workspace paths.
2. Writes a placeholder `ssh_config` under SSHConfigDir.
3. Grouping/leaf Setup set Mode, PairName, seeds, Exec fakes, and probe hooks.
4. Root `Run` applies seeds then dispatches BuildUnisonCmd / RunPair / RunCLI.

## Context

- Module: `github.com/xhd2015/ai-critic`
- Tree home: `cmd/agentcli/synccmd/run_tests`
- Spec version 0.0.2; L2 in-process only.
- P1 `sync_tests/` and P2 `doctor_status_tests/` remain sealed and GREEN.

```go
import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	if req.StoreDir == "" {
		req.StoreDir = filepath.Join(d.DOCTEST_CASE, "store")
	}
	if req.UnisonDir == "" {
		req.UnisonDir = filepath.Join(d.DOCTEST_CASE, "unison")
	}
	if req.SSHConfigDir == "" {
		req.SSHConfigDir = filepath.Join(d.DOCTEST_CASE, "ssh")
	}
	if req.LocalPath == "" {
		req.LocalPath = filepath.Join(d.DOCTEST_CASE, "workspace-local")
	}
	if req.RemotePath == "" {
		req.RemotePath = filepath.Join(d.DOCTEST_CASE, "workspace-remote")
	}
	for _, dir := range []string{req.StoreDir, req.UnisonDir, req.SSHConfigDir, req.LocalPath, req.RemotePath} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	sshCfg := filepath.Join(req.SSHConfigDir, "ssh_config")
	if _, err := os.Stat(sshCfg); os.IsNotExist(err) {
		if err := os.WriteFile(sshCfg, []byte("# test ssh_config\nHost remote-agent\n"), 0o644); err != nil {
			return err
		}
	}
	return nil
}
```
