# Scenario

**Feature**: status on seeded session shows multi-layer probe

```
seed sess-st + ProbeInject -> agent-run status --json sess-st
  -> session/process/terminal/runner/resume fields
```

## Preconditions

- ProbeInject supplies L2 fake layers (no real TTY).
- Prefer `--json` for stable field checks.

## Steps

1. Seed sess-st; set ProbeInject; CLI status --json sess-st.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "status", "--json", "sess-st")
	req.Seeds = seedBoundExited("sess-st", "grok-run-1", "term-st")
	exited := true
	req.ProbeInject = &StatusProbeSeed{
		Session:        "grok/sess-st",
		Status:         "finished",
		Workspace:      "/tmp/ws-resume",
		ProcessStatus:  "dead",
		TerminalStatus: "unreachable",
		RunnerStatus:   "bound",
		RunnerExited:   &exited,
		ResumeReady:    true,
		ResumeReason:   "bound and exited",
	}
	return nil
}
```
