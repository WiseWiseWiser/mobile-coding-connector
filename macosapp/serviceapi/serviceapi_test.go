package serviceapi

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAuthorizationHeader(t *testing.T) {
	if got := AuthorizationHeader("abc"); got != "Bearer abc" {
		t.Fatalf("got %q", got)
	}
	if got := AuthorizationHeader(""); got != "" {
		t.Fatalf("empty token should omit header, got %q", got)
	}
}

func TestListServicesPath(t *testing.T) {
	if got := ListServicesPath(); got != "/api/services?all=1" {
		t.Fatalf("got %q", got)
	}
}

func TestServiceActionPath_encodesID(t *testing.T) {
	got := ServiceActionPath(ActionStart, "svc with space")
	if !strings.HasPrefix(got, "/api/services/start?") {
		t.Fatalf("path = %q", got)
	}
	if !strings.Contains(got, "id=svc") {
		t.Fatalf("missing id: %q", got)
	}
}

func TestBuildListServicesRequest_withToken(t *testing.T) {
	req, err := BuildListServicesRequest("https://agent.example.com/", "secret-token")
	if err != nil {
		t.Fatal(err)
	}
	if req.Method != "GET" {
		t.Fatalf("method = %q", req.Method)
	}
	if req.URL != "https://agent.example.com/api/services?all=1" {
		t.Fatalf("url = %q", req.URL)
	}
	if req.Headers["Authorization"] != "Bearer secret-token" {
		t.Fatalf("auth = %q", req.Headers["Authorization"])
	}
	// must not use loopback keep-alive
	if strings.Contains(req.URL, "127.0.0.1") || strings.Contains(req.URL, "23312") {
		t.Fatalf("must not target keep-alive: %q", req.URL)
	}
}

func TestBuildListServicesRequest_emptyToken_noAuthHeader(t *testing.T) {
	req, err := BuildListServicesRequest("https://agent.example.com", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := req.Headers["Authorization"]; ok {
		t.Fatalf("unexpected Authorization: %v", req.Headers)
	}
}

func TestBuildServiceActionRequest_startStopRestartEnableDisable(t *testing.T) {
	base := "https://remote.example"
	token := "tok"
	cases := []struct {
		action ServiceAction
		want   string
	}{
		{ActionStart, "/api/services/start?id=svc-1"},
		{ActionStop, "/api/services/stop?id=svc-1"},
		{ActionRestart, "/api/services/restart?id=svc-1"},
		{ActionEnable, "/api/services/enable?id=svc-1"},
		{ActionDisable, "/api/services/disable?id=svc-1"},
	}
	for _, tc := range cases {
		req, err := BuildServiceActionRequest(base, token, tc.action, "svc-1")
		if err != nil {
			t.Fatalf("%s: %v", tc.action, err)
		}
		if req.Method != "POST" {
			t.Fatalf("%s method = %q", tc.action, req.Method)
		}
		if req.URL != base+tc.want {
			t.Fatalf("%s url = %q want %q", tc.action, req.URL, base+tc.want)
		}
		if req.Headers["Authorization"] != "Bearer tok" {
			t.Fatalf("%s auth missing", tc.action)
		}
	}
}

func TestBuildServiceActionRequest_requiresBaseAndID(t *testing.T) {
	if _, err := BuildServiceActionRequest("", "t", ActionStart, "id"); err == nil {
		t.Fatal("expected error for empty base")
	}
	if _, err := BuildServiceActionRequest("https://x.com", "t", ActionStart, ""); err == nil {
		t.Fatal("expected error for empty id")
	}
}

// Server body shapes (see server/services handlers):
// start → ServiceStatus; stop/restart → {"status":"ok"}; enable/disable → {status,message,service?}
func TestAcceptServiceActionBody_serverShapes(t *testing.T) {
	startBody := []byte(`{
  "id": "svc-1",
  "name": "demo",
  "status": "running",
  "pid": 42,
  "logPath": "/tmp/demo.log",
  "desiredRunning": true,
  "enabled": true
}`)
	if _, ok := AcceptServiceActionBody(startBody); !ok {
		t.Fatal("start ServiceStatus body must be accepted after HTTP 200")
	}

	stopBody := []byte(`{"status":"ok"}`)
	if _, ok := AcceptServiceActionBody(stopBody); !ok {
		t.Fatal("stop/restart {status:ok} body must be accepted")
	}

	enableBody := []byte(`{
  "status": "ok",
  "message": "The server won't start immediately until daemon checks at next time",
  "service": {
    "id": "svc-1",
    "name": "demo",
    "status": "stopped",
    "pid": 0,
    "logPath": "/tmp/demo.log",
    "desiredRunning": false,
    "enabled": true
  }
}`)
	msg, ok := AcceptServiceActionBody(enableBody)
	if !ok {
		t.Fatal("enable/disable body must be accepted")
	}
	if msg == "" {
		t.Fatal("enable body should expose message")
	}

	// Go's encoding/json fills missing string fields with ""; Swift non-optional
	// Codable message fails on start/stop bodies. AcceptServiceActionBody is the
	// shared success gate; Swift ServiceClient.decodeServiceActionBody mirrors it.
	type loose struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	var looseStart loose
	if err := json.Unmarshal(startBody, &looseStart); err != nil {
		t.Fatalf("unexpected go unmarshal: %v", err)
	}
	if looseStart.Message != "" {
		t.Fatalf("start body should not carry message, got %q", looseStart.Message)
	}
	if _, ok := AcceptServiceActionBody([]byte(`not-json`)); ok {
		t.Fatal("invalid JSON must not be accepted")
	}
}
