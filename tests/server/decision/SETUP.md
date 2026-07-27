# Scenario

**Feature**: decision

```
decision
```

## Preconditions

This grouping node covers auto-start behaviour of the server. Future children
may test additional auto-start scenarios (e.g., disabled, localhost domain,
missing binary, auth proxy).

The shared precondition for all children under this node is that the server
is started with a custom config home directory via `AI_CRITIC_HOME`.

## Steps

1. The root `Run` function has already set up the config home and built/started
   the server.
2. Child `Setup` functions configure the specific opencode settings file for
   their scenario.
3. Child `Assert` functions verify the expected behaviour.

```go
import (
	"os"
	"testing"

	"github.com/xhd2015/ai-critic/script/lib"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	if req.ConfigHome == "" {
		configHome, err := lib.CreateTestConfigHome()
		if err != nil {
			return err
		}
		t.Logf("created config home: %s", configHome)
		t.Cleanup(func() {
			os.RemoveAll(configHome)
		})
		req.ConfigHome = configHome
	}
	return nil
}
```
