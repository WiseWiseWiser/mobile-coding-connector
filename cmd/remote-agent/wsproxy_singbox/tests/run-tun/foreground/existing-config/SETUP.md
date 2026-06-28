# Scenario

**Feature**: run-tun uses user-supplied `--config` file

```
# --config FILE: skip FetchVMess, use existing sing-box JSON
--config FILE -> RunSingBox (no API fetch)
```

## Preconditions

- `SingBoxOnPath = true` so execution reaches RunSingBox.

```go
import (
	"os"
	"path/filepath"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	req.SingBoxOnPath = true
	req.IsTTY = true
	req.EUID = euidPtr(1000)
	cfgPath := filepath.Join("existing-singbox.json")
	cfg := `{"inbounds":[{"type":"tun"}],"outbounds":[{"type":"direct","tag":"direct"}]}`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0600); err != nil {
		return err
	}
	req.ConfigFile = cfgPath
	return nil
}
```