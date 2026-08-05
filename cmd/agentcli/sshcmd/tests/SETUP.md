# Scenario

**Feature**: remote-agent ssh P1 — parse modes, dest strip, session gate, injectables

```
# operator argv after `ssh` subcommand
operator -> sshcmd.Parse(args) -> Mode + Dest + RemoteArgv
operator -> sshcmd.Run(Options{Store,Serve,Runner,Stdout,Stderr})
  -> help: Usage on Stdout
  -> serve: ServeStarter.Start (exclusive)
  -> login/command: SessionStore.Load; gate Alive; SSHRunner.Run(remoteArgv)
```

## Preconditions

- Intended package: `github.com/xhd2015/ai-critic/cmd/agentcli/sshcmd` (may be
  missing until implementer — suite is **RED** on compile/link).
- Public surface under test:
  - `Parse(args []string) (*ParseResult, error)`
  - `Run(opts Options) error` with injectable `SessionStore`, `ServeStarter`,
    `SSHRunner`, and `Stdout`/`Stderr` writers
  - Types: `Mode`, `ParseResult`, `Session`, `Options`, `ServeOpts`, `RunnerOpts`
- Destination strip matcher (strict): `^[^@/\s]+@[^@/\s]+$`
- Client no-session error (exact substring contract):
  `no active SSH tunnel; run 'remote-agent ssh --serve' first`
- Help stdout ends with trailing `\n` (POSIX).
- No real SSH, network, or process env/cwd mutation.
- Parallel-safe: mocks and writers only via `Request` / `Options`; use `d` for
  any paths (none required for pure L2 leaves).

## Steps

1. Root Setup normalizes `Args` to non-nil and default `ProfileID` to `"default"`.
2. Mode grouping Setup narrows behavioral mode (help | serve | login | command).
3. Leaf Setup sets concrete `Args` and optional `Session` mock state.
4. Root `Run` calls `sshcmd.Parse` then `sshcmd.Run` with mock store/serve/runner
   and buffer writers; records call counts and remote argv.

## Context

- Module: `github.com/xhd2015/ai-critic`
- Tree home: `cmd/agentcli/sshcmd/tests` (alongside `streamcmd/tests`)
- L2 in-process only; no L3 e2e binary.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	if req.Args == nil {
		req.Args = []string{}
	}
	if req.ProfileID == "" {
		req.ProfileID = "default"
	}
	return nil
}
```
