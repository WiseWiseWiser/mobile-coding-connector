# Scenario

**Feature**: `--config` path passed through to sing-box run

```
# user config file used as-is for sing-box -c
RunSingBox(configPath=user --config FILE)
```

## Steps

1. Parent `existing-config/SETUP` seeds `existing-singbox.json`.

```go
import (
	"os"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	if _, err := os.Stat(req.ConfigFile); err != nil {
		t.Fatalf("precondition: config file missing: %v", err)
	}
	return nil
}
```