# Scenario

**Feature**: remote-agent sync unison P1 — pair store CRUD + Unison profile emit

```
# operator argv after `sync`
operator -> synccmd.RunCLI(args, CLIOpts{StoreDir,UnisonDir,SSHConfigDir,Stdout,Stderr})
  -> help | unison init|add|list|show|set|rm
  -> {StoreDir}/pairs.json
  -> {UnisonDir}/remote-agent-<name>.prf  (sshargs -F {SSHConfigDir}/ssh_config)
```

## Preconditions

- Intended package: `github.com/xhd2015/ai-critic/cmd/agentcli/synccmd` (may be
  missing until implementer — suite is **RED** on compile/link).
- Public surface under test: `RunCLI`, `Store`, `InitPair`/`SetPair`/`RmPair`,
  `WriteUnisonProfile`/`RenderUnisonProfile`/`ProfileFileName`, types `Pair`,
  `Config`, `CLIOpts`, opts structs.
- Injectable absolute `StoreDir`, `UnisonDir`, `SSHConfigDir` and writers only —
  no `os.UserHomeDir` hard dependency in library core for tests.
- Parallel-safe: paths under `d.DOCTEST_CASE`; no `Setenv`/`Chdir`.
- No real Unison binary, SSH, or network.

## Steps

1. Root Setup assigns absolute StoreDir / UnisonDir / SSHConfigDir under
   `d.DOCTEST_CASE` when empty; creates those directories.
2. Optionally sets default LocalPath / RemotePath workspace dirs under the case
   when empty (for init leaves).
3. Grouping Setup narrows operation family and outcome class.
4. Leaf Setup sets concrete `Args` / `PreArgs` / `FocusPair` / seeds.
5. Root `Run` executes PreArgs then Args via `RunCLI`, then loads store + profile.

## Context

- Module: `github.com/xhd2015/ai-critic`
- Tree home: `cmd/agentcli/synccmd/sync_tests`
- Spec version 0.0.2; L2 in-process only.

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
	// Placeholder ssh_config so profile path is a real file location under case.
	sshCfg := filepath.Join(req.SSHConfigDir, "ssh_config")
	if _, err := os.Stat(sshCfg); os.IsNotExist(err) {
		if err := os.WriteFile(sshCfg, []byte("# test ssh_config\nHost remote-agent\n"), 0o644); err != nil {
			return err
		}
	}
	return nil
}
```
