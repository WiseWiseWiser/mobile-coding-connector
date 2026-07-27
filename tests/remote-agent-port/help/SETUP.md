# Scenario

**Feature**: help text for port subcommands

```
remote-agent port [--help] | port list --help | port visit --help -> usage on stdout
```

## Preconditions

CLI help does not require a live server for content (L2 still starts mux).

## Steps

1. Leaf sets `Op=cli` and help Args.
2. Run agentcli; Assert matches usage templates.

## Context

Help surfaces mirror agentcli flag style (`less-gen/flags`).

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "cli"
	return nil
}
```
