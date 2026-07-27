# Scenario

**Feature**: remote app profile does not spawn daemon

```
appprofile.Remote() -> SpawnsDaemon=false, UsesAuthToken=true,
  ConfigFileName=remote-agent-config.json, display contains Remote,
  AppName=ai-critic-remote-macos, BundleID=com.xhd2015.ai-critic-remote-macos
```

## Preconditions

Product identity table from requirement.

## Steps

1. Set `ProfileName=remote`.

## Context

REQUIREMENT leaf: `profile/remote`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ProfileName = "remote"
	return nil
}
```
