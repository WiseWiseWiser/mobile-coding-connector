# Scenario

**Feature**: remote-agent sync unison P2 — doctor + status with injectable probes

```
# library path
operator -> synccmd.Doctor|Status(opts{StoreDir, probes}) -> report

# CLI path
operator -> synccmd.RunCLI([unison doctor|status ...], CLIOpts{probes})
  -> stdout table / status lines
  -> {StoreDir}/pairs.json + optional state/<name>.json
```

## Preconditions

- Package: `github.com/xhd2015/ai-critic/cmd/agentcli/synccmd` (P1 GREEN; P2 APIs
  may be missing → suite **RED** on compile/link until implementer).
- Public surface under test: `Doctor`, `Status`, extended `CLIOpts` / `RunCLI`,
  types `DoctorOpts`, `DoctorReport`, `DoctorCheck`, `StatusOpts`, `StatusReport`,
  `StatusItem`.
- Injectable absolute dirs + probe hooks only — no process env/cwd mutation.
- Parallel-safe: paths under `d.DOCTEST_CASE`.
- No real Unison binary, SSH, or network.

## Steps

1. Root Setup assigns absolute StoreDir / UnisonDir / SSHConfigDir under
   `d.DOCTEST_CASE` when empty; creates those directories and workspace paths.
2. Writes a placeholder `ssh_config` under SSHConfigDir.
3. Defaults EnsureLocalRoot to true when leaf did not set EnsureLocalRootSet.
4. Grouping/leaf Setup set Mode, PairName, seeds, and probe fakes.
5. Root `Run` applies seeds then dispatches Doctor / Status / RunCLI.

## Context

- Module: `github.com/xhd2015/ai-critic`
- Tree home: `cmd/agentcli/synccmd/doctor_status_tests`
- Spec version 0.0.2; L2 in-process only.
- P1 tree `sync_tests/` remains sealed and GREEN independently.

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
	if !req.EnsureLocalRootSet {
		req.EnsureLocalRoot = true
		req.EnsureLocalRootSet = true
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
