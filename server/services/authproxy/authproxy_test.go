package authproxy

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestHandlerLoginAndProxy(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := r.Cookie(defaultCookieName); err == nil {
			t.Errorf("backend received auth cookie")
		}
		w.Write([]byte("ok:" + r.URL.Path))
	}))
	defer backend.Close()

	h := mustHandler(t, Config{
		BackendURL:  backend.URL,
		TokenMode:   TokenModeCustom,
		CustomToken: "s3cret",
		Secret:      bytes.Repeat([]byte("s"), 32),
	})
	srv := httptest.NewServer(h)
	defer srv.Close()

	client := srv.Client()
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}

	resp, err := client.Get(srv.URL + "/app")
	if err != nil {
		t.Fatalf("GET /app: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("unauthenticated status = %d, want 302", resp.StatusCode)
	}
	location := resp.Header.Get("Location")
	if !strings.HasPrefix(location, loginPath) || !strings.Contains(location, "next="+url.QueryEscape("/app")) {
		t.Fatalf("Location = %q, want login with next=/app", location)
	}

	login := postJSON(t, client, srv.URL+loginPath, map[string]string{
		"username": "ada",
		"token":    "s3cret",
	})
	defer login.Body.Close()
	if login.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, want 200", login.StatusCode)
	}
	cookie := sessionCookie(login)
	if cookie == "" {
		t.Fatal("login did not set cookie")
	}

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/app", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Cookie", defaultCookieName+"="+cookie)
	proxied, err := client.Do(req)
	if err != nil {
		t.Fatalf("authenticated GET: %v", err)
	}
	body, _ := io.ReadAll(proxied.Body)
	proxied.Body.Close()
	if proxied.StatusCode != http.StatusOK || string(body) != "ok:/app" {
		t.Fatalf("proxied = %d %q, want 200 ok:/app", proxied.StatusCode, body)
	}
}

func TestHandlerWrongCookieGoesToExpired(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("backend should not be reached")
	}))
	defer backend.Close()

	h := mustHandler(t, Config{
		BackendURL:  backend.URL,
		TokenMode:   TokenModeCustom,
		CustomToken: "s3cret",
		Secret:      bytes.Repeat([]byte("s"), 32),
	})
	srv := httptest.NewServer(h)
	defer srv.Close()
	client := noRedirectClient(srv)

	login := postJSON(t, client, srv.URL+loginPath, map[string]string{
		"username": "ada",
		"token":    "s3cret",
	})
	cookie := sessionCookie(login)
	login.Body.Close()

	h2 := mustHandler(t, Config{
		BackendURL:  backend.URL,
		TokenMode:   TokenModeCustom,
		CustomToken: "rotated",
		Secret:      bytes.Repeat([]byte("s"), 32),
	})
	srv2 := httptest.NewServer(h2)
	defer srv2.Close()
	client2 := noRedirectClient(srv2)

	req, err := http.NewRequest(http.MethodGet, srv2.URL+"/app", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Cookie", defaultCookieName+"="+cookie)
	resp, err := client2.Do(req)
	if err != nil {
		t.Fatalf("GET with rotated token: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusFound || resp.Header.Get("Location") != expiredPath {
		t.Fatalf("status=%d location=%q, want 302 %s", resp.StatusCode, resp.Header.Get("Location"), expiredPath)
	}
}

func TestHandlerExpiredCookieGoesToLogin(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer backend.Close()

	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	h := mustHandler(t, Config{
		BackendURL:  backend.URL,
		TokenMode:   TokenModeCustom,
		CustomToken: "s3cret",
		Secret:      bytes.Repeat([]byte("s"), 32),
		TTL:         time.Hour,
		Now:         func() time.Time { return now },
	})
	srv := httptest.NewServer(h)
	defer srv.Close()
	client := noRedirectClient(srv)

	login := postJSON(t, client, srv.URL+loginPath, map[string]string{
		"username": "ada",
		"token":    "s3cret",
	})
	cookie := sessionCookie(login)
	login.Body.Close()

	later := now.Add(2 * time.Hour)
	h2 := mustHandler(t, Config{
		BackendURL:  backend.URL,
		TokenMode:   TokenModeCustom,
		CustomToken: "s3cret",
		Secret:      bytes.Repeat([]byte("s"), 32),
		TTL:         time.Hour,
		Now:         func() time.Time { return later },
	})
	srv2 := httptest.NewServer(h2)
	defer srv2.Close()
	client2 := noRedirectClient(srv2)

	req, err := http.NewRequest(http.MethodGet, srv2.URL+"/app", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Cookie", defaultCookieName+"="+cookie)
	resp, err := client2.Do(req)
	if err != nil {
		t.Fatalf("GET expired cookie: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusFound || !strings.HasPrefix(resp.Header.Get("Location"), loginPath) {
		t.Fatalf("status=%d location=%q, want 302 login", resp.StatusCode, resp.Header.Get("Location"))
	}
}

func TestHandlerFixedUserAndSharedToken(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer backend.Close()

	h := mustHandler(t, Config{
		BackendURL: backend.URL,
		AuthUser:   "alice",
		TokenMode:  TokenModeShared,
		Secret:     bytes.Repeat([]byte("s"), 32),
		LookupShared: func() ([]string, error) {
			return []string{"shared-token"}, nil
		},
	})
	srv := httptest.NewServer(h)
	defer srv.Close()
	client := noRedirectClient(srv)

	badUser := postJSON(t, client, srv.URL+loginPath, map[string]string{
		"username": "bob",
		"token":    "shared-token",
	})
	body, _ := io.ReadAll(badUser.Body)
	badUser.Body.Close()
	if badUser.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong user status = %d body %s, want 401", badUser.StatusCode, body)
	}

	okLogin := postJSON(t, client, srv.URL+loginPath, map[string]string{
		"username": "alice",
		"token":    "shared-token",
	})
	okLogin.Body.Close()
	if okLogin.StatusCode != http.StatusOK {
		t.Fatalf("alice login status = %d, want 200", okLogin.StatusCode)
	}
}

func TestHandlerBearerAndWebSocket(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
			http.Error(w, "not ws", http.StatusBadRequest)
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		_ = conn.WriteMessage(websocket.TextMessage, msg)
	}))
	defer backend.Close()

	h := mustHandler(t, Config{
		BackendURL:  backend.URL,
		TokenMode:   TokenModeCustom,
		CustomToken: "s3cret",
		Secret:      bytes.Repeat([]byte("s"), 32),
	})
	srv := httptest.NewServer(h)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	_, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("unauthenticated websocket dial succeeded")
	}

	header := http.Header{}
	header.Set("Authorization", "Bearer s3cret")
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("authenticated websocket dial: %v", err)
	}
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	defer conn.Close()
	if err := conn.WriteMessage(websocket.TextMessage, []byte("ping")); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(msg) != "ping" {
		t.Fatalf("ws echo = %q, want ping", msg)
	}
}

func TestHandlerReservedAuthPathNotProxied(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("backend-login"))
	}))
	defer backend.Close()

	h := mustHandler(t, Config{
		BackendURL:  backend.URL,
		TokenMode:   TokenModeCustom,
		CustomToken: "s3cret",
		Secret:      bytes.Repeat([]byte("s"), 32),
	})
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + loginPath)
	if err != nil {
		t.Fatalf("GET login: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), "backend-login") {
		t.Fatal("login path was forwarded to backend")
	}
	if !strings.Contains(string(body), "Sign in") {
		t.Fatalf("login page body = %q", body)
	}
}

func TestSafeNextPathRejectsOpenRedirect(t *testing.T) {
	if got := safeNextPath("https://evil.example/phish"); got != "/" {
		t.Fatalf("abs url = %q, want /", got)
	}
	if got := safeNextPath("//evil.example"); got != "/" {
		t.Fatalf("protocol-relative = %q, want /", got)
	}
	if got := safeNextPath("/ok?x=1"); got != "/ok?x=1" {
		t.Fatalf("relative = %q, want /ok?x=1", got)
	}
}

func TestLoadOrCreateSecretRoundTrip(t *testing.T) {
	path := t.TempDir() + "/service-auth-secret"
	first, err := LoadOrCreateSecret(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(first) != 32 {
		t.Fatalf("secret len = %d, want 32", len(first))
	}
	second, err := LoadOrCreateSecret(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("reloaded secret did not match")
	}
}

func mustHandler(t *testing.T, cfg Config) http.Handler {
	t.Helper()
	cfg = normalizeConfig(cfg)
	backend, err := cfg.backendURL()
	if err != nil {
		t.Fatal(err)
	}
	h, err := newHandler(cfg, backend)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func noRedirectClient(srv *httptest.Server) *http.Client {
	client := srv.Client()
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return client
}

func postJSON(t *testing.T, client *http.Client, url string, payload map[string]string) *http.Response {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func sessionCookie(resp *http.Response) string {
	for _, cookie := range resp.Cookies() {
		if cookie.Name == defaultCookieName {
			return cookie.Value
		}
	}
	return ""
}
