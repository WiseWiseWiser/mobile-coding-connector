package openclaw

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestManagerStartStopMockLifecycle(t *testing.T) {
	useTempDataDir(t)
	cfg := &Config{
		GatewayPort: 18789,
		Slack: &SlackConfig{
			Enabled:  true,
			BotToken: "xoxb-test",
			AppToken: "xapp-test",
		},
	}
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	m := &Manager{}
	if err := m.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	status := m.Status()
	if !status.Running || !status.Mocked || status.MockPID != mockPID {
		t.Fatalf("Status() after start = %+v", status)
	}

	if err := m.Start(); err == nil {
		t.Fatal("second Start() should fail")
	}

	if err := m.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	status = m.Status()
	if status.Running {
		t.Fatalf("Status() after stop = %+v", status)
	}
}

func TestAPIConfigMaskingAndStart(t *testing.T) {
	useTempDataDir(t)

	mux := http.NewServeMux()
	RegisterAPI(mux)

	putBody := `{"slack":{"enabled":true,"bot_token":"xoxb-secret","app_token":"xapp-secret"}}`
	putReq := httptest.NewRequest(http.MethodPut, "/api/openclaw/config", strings.NewReader(putBody))
	putRec := httptest.NewRecorder()
	mux.ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("PUT config status = %d body = %s", putRec.Code, putRec.Body.String())
	}
	if !contains(putRec.Body.String(), `"bot_token":"***"`) {
		t.Fatalf("PUT response should mask bot token: %s", putRec.Body.String())
	}

	startReq := httptest.NewRequest(http.MethodPost, "/api/openclaw/start", nil)
	startRec := httptest.NewRecorder()
	mux.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("POST start status = %d body = %s", startRec.Code, startRec.Body.String())
	}
	if !contains(startRec.Body.String(), `"running":true`) {
		t.Fatalf("start response = %s", startRec.Body.String())
	}

	statusReq := httptest.NewRequest(http.MethodGet, "/api/openclaw/status", nil)
	statusRec := httptest.NewRecorder()
	mux.ServeHTTP(statusRec, statusReq)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("GET status code = %d", statusRec.Code)
	}
	if !contains(statusRec.Body.String(), `"mocked":true`) {
		t.Fatalf("status response = %s", statusRec.Body.String())
	}
}

