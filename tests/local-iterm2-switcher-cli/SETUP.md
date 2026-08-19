# Scenario

**Feature**: local-agent native-terminals list/focus + inline split TUI (in-process)

```
# no --server / --token / daemon
agentcli.RunWithWriters(LocalProfile()) + testhooks HOME
  -> native-terminals list|focus|-h
localiterm2 file cache + Capture/Layout/Focus hooks (not HTTP)
Op=tui -> itermswitcher.View / ApplyKey (pure state)
--tty + SetTerminalsKeys -> scripted TUI without hijacking os.Stdin
```

## Preconditions

1. Isolated testhooks HOME per leaf; no process Setenv/Chdir.
2. List/focus leaves install Capture/Layout/ITermRunning/Focus under `agentcliMu`.
3. Help/dispatch leaves set `SkipHooks` (no inject needed).
4. Seeded last-good uses P1 fixture JSON (`sess-a`, cwd ai-critic).
5. TUI library leaves use `Op=tui`; CLI TUI uses `--tty` + `Keys`.

## Steps

1. Root Setup validates request pointer.
2. Leaf Setup sets Args / SeedCache / ITermDown / SkipHooks / Op / ApplyKeys / Keys.
3. Run captures stdout/stderr/exit, hook counters, or View/ApplyKey results.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	if req == nil {
		t.Fatal("nil request")
	}
	return nil
}
```
