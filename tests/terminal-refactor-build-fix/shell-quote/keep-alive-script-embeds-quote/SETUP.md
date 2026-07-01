# Scenario

**Feature**: keep-alive script embeds shell-quoted paths when bin/args contain spaces

```
# keep-alive script generation
run.TestExported_OutputKeepAliveScript -> script with quoted spaced paths -> sh -n OK
```

## Preconditions

- Implementer provides `run.TestExported_OutputKeepAliveScript(port, args, binPath, w)`.

```go
import (
	"path/filepath"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	req.Phase = "keep-alive-script"
	req.KeepAlivePort = 14099
	req.KeepAliveBinPath = filepath.Join(t.TempDir(), "ai critic", "ai-critic")
	req.KeepAliveServerArgs = []string{
		"--config",
		filepath.Join(t.TempDir(), "my config", "settings.json"),
	}
	return nil
}
```