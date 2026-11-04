package httpproxy

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func startProxy(t *testing.T) (proxyURL *url.URL, cfg *Config) {
	t.Helper()
	SetConfigDir(t.TempDir())
	cfg = defaultConfig()
	cfg.Username = "tester"
	cfg.Password = "secret-pw"
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
	if err := GetManager().Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	st := GetManager().Status()
	if !st.Running {
		t.Fatal("proxy not running after start")
	}
	proxyURL = &url.URL{Scheme: "http", Host: fmt.Sprintf("127.0.0.1:%d", st.Port)}
	t.Cleanup(func() { _ = GetManager().Stop() })
	return proxyURL, cfg
}

func proxyAuthHeader(user, pass string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+pass))
}

func TestProxyRequiresAuth(t *testing.T) {
	proxyURL, _ := startProxy(t)

	req, _ := http.NewRequest("GET", "http://example.test/hello", nil)
	tr := &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatalf("roundtrip: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusProxyAuthRequired {
		t.Fatalf("status = %d, want 407", resp.StatusCode)
	}
	if resp.Header.Get("Proxy-Authenticate") == "" {
		t.Fatal("missing Proxy-Authenticate challenge")
	}
}

func TestProxyForwardsAbsoluteForm(t *testing.T) {
	proxyURL, cfg := startProxy(t)

	var gotPath, gotAuth string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Proxy-Authorization")
		fmt.Fprint(w, "hello-from-backend")
	}))
	defer backend.Close()

	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
	req, _ := http.NewRequest("GET", backend.URL+"/some/path", nil)
	req.SetBasicAuth(cfg.Username, cfg.Password)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || string(body) != "hello-from-backend" {
		t.Fatalf("status=%d body=%q", resp.StatusCode, string(body))
	}
	if gotPath != "/some/path" {
		t.Fatalf("backend path = %q", gotPath)
	}
	if gotAuth != "" {
		t.Fatal("Proxy-Authorization must not leak to the backend")
	}

	items, active, _, total, _ := GetManager().Connections()
	if total < 1 {
		t.Fatalf("total connections = %d", total)
	}
	if active != 0 {
		t.Fatalf("active = %d after completion", active)
	}
	if len(items) == 0 || items[0].Method != "GET" {
		t.Fatalf("items = %+v", items)
	}
}

func TestProxyConnectTunnel(t *testing.T) {
	proxyURL, cfg := startProxy(t)

	// Loopback targets are refused by default; this tunnel test dials a local
	// echo server, so opt in.
	cfg.AllowLoopbackConnect = true
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("save: %v", err)
	}
	// Re-read config into the live manager (stop/start cycle).
	if err := GetManager().Stop(); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if err := GetManager().Start(); err != nil {
		t.Fatalf("restart: %v", err)
	}
	st := GetManager().Status()
	proxyURL.Host = fmt.Sprintf("127.0.0.1:%d", st.Port)

	// Raw TCP echo standing in for an arbitrary host:port target.
	echoLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer echoLn.Close()
	go func() {
		for {
			conn, err := echoLn.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				_, _ = io.Copy(c, c)
				_ = c.Close()
			}(conn)
		}
	}()
	target := echoLn.Addr().String()

	conn, err := net.DialTimeout("tcp", proxyURL.Host, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	fmt.Fprintf(conn, "CONNECT %s HTTP/1.1\r\nHost: %s\r\nProxy-Authorization: %s\r\n\r\n",
		target, target, proxyAuthHeader(cfg.Username, cfg.Password))

	br := bufio.NewReader(conn)
	statusLine, err := br.ReadString('\n')
	if err != nil {
		t.Fatalf("read 200: %v", err)
	}
	if !strings.Contains(statusLine, "200") {
		t.Fatalf("CONNECT status = %q", statusLine)
	}
	// Drain the rest of the response head.
	for {
		line, err := br.ReadString('\n')
		if err != nil || line == "\r\n" {
			break
		}
	}

	fmt.Fprint(conn, "ping-through-tunnel")
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	reply, err := io.ReadAll(conn)
	if err != nil && len(reply) == 0 {
		t.Fatalf("tunnel echo: %v", err)
	}
	if string(reply) != "ping-through-tunnel" {
		t.Fatalf("tunnel reply = %q", string(reply))
	}

	_, _, connectActive, _, _ := GetManager().Connections()
	_ = connectActive
}

func TestProxyConnectRejectsLoopback(t *testing.T) {
	proxyURL, cfg := startProxy(t)

	conn, err := net.DialTimeout("tcp", proxyURL.Host, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	fmt.Fprintf(conn, "CONNECT 127.0.0.1:22 HTTP/1.1\r\nHost: 127.0.0.1:22\r\nProxy-Authorization: %s\r\n\r\n",
		proxyAuthHeader(cfg.Username, cfg.Password))
	br := bufio.NewReader(conn)
	_ = br
	statusLine, err := br.ReadString('\n')
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(statusLine, "403") {
		t.Fatalf("loopback CONNECT status = %q, want 403", statusLine)
	}
}

func TestProxyLockoutAfterFailures(t *testing.T) {
	proxyURL, cfg := startProxy(t)

	proxyClient := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
	bad := func() int {
		req, _ := http.NewRequest("GET", "http://example.test/", nil)
		req.SetBasicAuth(cfg.Username, "wrong-password")
		resp, err := proxyClient.Do(req)
		if err != nil {
			t.Fatalf("do: %v", err)
		}
		defer resp.Body.Close()
		return resp.StatusCode
	}

	saw429 := false
	for i := 0; i < cfg.FailLimit+3; i++ {
		code := bad()
		if code == http.StatusTooManyRequests {
			saw429 = true
			break
		}
		if code != http.StatusProxyAuthRequired {
			t.Fatalf("attempt %d: status = %d, want 407", i, code)
		}
	}
	if !saw429 {
		t.Fatal("lockout never triggered 429")
	}

	// Correct credentials are still refused while locked (same window).
	req, _ := http.NewRequest("GET", "http://example.test/", nil)
	req.SetBasicAuth(cfg.Username, cfg.Password)
	resp, err := proxyClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("locked status = %d, want 429", resp.StatusCode)
	}
}

func TestRotatePasswordChangesCredential(t *testing.T) {
	proxyURL, cfg := startProxy(t)

	newPW, err := GetManager().RotatePassword()
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if newPW == "" || newPW == cfg.Password {
		t.Fatalf("rotate returned %q (old %q)", newPW, cfg.Password)
	}

	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
	try := func(pw string) int {
		req, _ := http.NewRequest("GET", "http://example.test/", nil)
		req.SetBasicAuth(cfg.Username, pw)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("do: %v", err)
		}
		defer resp.Body.Close()
		return resp.StatusCode
	}
	if code := try(cfg.Password); code != http.StatusProxyAuthRequired {
		t.Fatalf("old password still accepted: %d", code)
	}
	// The new password must authenticate. example.test is unreachable, so a
	// 502 from the upstream dial proves auth passed; 407 would mean auth failed.
	if code := try(newPW); code == http.StatusProxyAuthRequired {
		t.Fatalf("new password rejected: %d", code)
	}
}
