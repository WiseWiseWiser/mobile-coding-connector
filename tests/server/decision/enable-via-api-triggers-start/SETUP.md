## Preconditions

1. A temporary config home directory exists
2. `AI_CRITIC_HOME` is set to the temp directory
3. `opencode.json` is written to the config home with:
   - `WebServer.Enabled = false`
   - `WebServer.Port = 14096`
   - `DefaultDomain = "test-enable-via-api.example.com"`
   - `AuthProxyEnabled = false`
4. The server binary is built
5. The server is started in normal (non-quick-test) mode with auto-start disabled

## Steps

1. Set `Request.OpenCodeSettings` with `WebServerEnabled=false`, a valid `DefaultDomain`, and `WebServerPort=14096`
2. Write `opencode.json` with `enabled: false` to the config home
3. Configure `Request.PostStart` to make a `POST` request to `/api/agents/opencode/settings` enabling the web server (`"web_server":{"enabled":true,...}`)
4. The root `Run` function:
   a. Builds and starts the server
   b. Waits for `/ping` to respond
   c. Sleeps 3 seconds for initial startup tasks
   d. Snapshots the current logs into `PrePostLogs`
   e. Makes the settings API call to enable the web server
   f. Waits 5 seconds for `AutoStartWebServer` to trigger and web server to start
   g. Checks port accessibility, stops server, captures full logs

## Context

This leaf tests that enabling the web server via the settings API triggers
`AutoStartWebServer()` even when the server was originally started with
`WebServer.Enabled=false`. This is the bug fix scenario: `handleOpencodeSettings`
should call `AutoStartWebServer()` after saving settings when the web server
becomes enabled.

The expected behaviour is:
- Initial startup: no auto-start log messages (since `Enabled=false`)
- After API call: auto-start log messages appear (`[opencode] AutoStartWebServer:`)
- If the `opencode` binary is available: the web server starts on port 14096
```

```go
import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	req.OpenCodeSettings = &OpenCodeSettings{
		WebServerEnabled: false,
		WebServerPort:    14096,
		DefaultDomain:    "test-enable-via-api.example.com",
	}

	configHome := os.Getenv("AI_CRITIC_HOME")
	if configHome == "" {
		var err error
		configHome, err = os.MkdirTemp("", "ai-critic-test-*")
		if err != nil {
			return fmt.Errorf("failed to create temp config home: %w", err)
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

	if err := os.MkdirAll(configHome, 0755); err != nil {
		return fmt.Errorf("failed to ensure config home dir: %w", err)
	}

	settings := map[string]interface{}{
		"default_domain": req.OpenCodeSettings.DefaultDomain,
		"web_server": map[string]interface{}{
			"enabled":            false,
			"port":               req.OpenCodeSettings.WebServerPort,
			"auth_proxy_enabled": false,
		},
	}

	opencodeJSON := filepath.Join(configHome, "opencode.json")
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal opencode settings: %w", err)
	}
	if err := os.WriteFile(opencodeJSON, data, 0644); err != nil {
		return fmt.Errorf("failed to write opencode.json: %w", err)
	}
	t.Logf("wrote opencode.json to %s: %s", opencodeJSON, string(data))

	req.PostStart = &PostStartRequest{
		URL:    "http://localhost:__PORT__/api/agents/opencode/settings",
		Method: "POST",
		Body:   `{"default_domain":"test-enable-via-api.example.com","web_server":{"enabled":true,"port":14096,"auth_proxy_enabled":false,"target_preference":"domain"}}`,
		Wait:   5,
	}

	if req.ServerPort <= 0 {
		req.ServerPort = 23712
	}
	if req.TimeoutSecs <= 0 {
		req.TimeoutSecs = 30
	}

	return nil
}
```
