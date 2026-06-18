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
)

func Setup(t *testing.T, req *Request) error {
	configHome := os.Getenv("AI_CRITIC_HOME")
	if configHome == "" {
		var err error
		configHome, err = os.MkdirTemp("", "ai-critic-test-*")
		if err != nil {
			return err
		}
		t.Logf("created config home: %s", configHome)
		t.Cleanup(func() {
			os.RemoveAll(configHome)
		})
		os.Setenv("AI_CRITIC_HOME", configHome)
		t.Cleanup(func() {
			os.Unsetenv("AI_CRITIC_HOME")
		})
	}
	return nil
}
```
