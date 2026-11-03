package authproxy

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/server/auth"
	"github.com/xhd2015/ai-critic/server/config"
	"github.com/xhd2015/ai-critic/server/proxy/origin"
)

func TestCustomManagementCookiePreservesUpstreamBrowserLogin(t *testing.T) {
	previous := auth.CookieName()
	if err := auth.SetCookieName("pi-agent-token"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = auth.SetCookieName(previous); auth.SetCredentialsFile(config.CredentialsFile) })
	credentials := filepath.Join(t.TempDir(), "pi-credentials")
	if err := os.WriteFile(credentials, []byte("fixture-pi-token\n"), 0600); err != nil {
		t.Fatal(err)
	}
	auth.SetCredentialsFile(credentials)
	// The upstream owns its own fixture credential, not auth's Pi-global state.
	// This models a separately running, unchanged default-name Mac instance.
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := r.Cookie("pi-agent-token"); err == nil {
			t.Error("Pi cookie reached upstream")
		}
		if _, err := r.Cookie(auth.ServiceCookieName); err == nil {
			t.Error("mapping gate cookie reached upstream")
		}
		if r.URL.Path == "/api/login" {
			var body struct{ Username, Password string }
			if r.Method != "POST" || json.NewDecoder(r.Body).Decode(&body) != nil || body.Password != "fixture-mac-token" {
				w.WriteHeader(401)
				return
			}
			http.SetCookie(w, &http.Cookie{Name: auth.DefaultCookieName, Value: body.Password, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: 3600})
			io.WriteString(w, `{"status":"ok"}`)
			return
		}
		cookie, _ := r.Cookie(auth.DefaultCookieName)
		if (cookie == nil || cookie.Value != "fixture-mac-token") && r.Header.Get("Authorization") != "Bearer fixture-mac-token" {
			w.WriteHeader(401)
			return
		}
		io.WriteString(w, `{"status":"ok"}`)
	}))
	defer upstream.Close()
	tr := origin.NewTransport()
	proxy, err := Start(Config{DisableAuth: true, BackendURL: upstream.URL, Transport: tr})
	if err != nil {
		t.Fatal(err)
	}
	defer proxy.Close()
	public, _ := url.Parse("http://127.0.0.1:" + strconv.Itoa(proxy.Port()))
	jar, _ := cookiejar.New(nil)
	jar.SetCookies(public, []*http.Cookie{{Name: "pi-agent-token", Value: "fixture-pi-token", Path: "/"}, {Name: auth.ServiceCookieName, Value: "gate-fixture", Path: "/"}})
	client := &http.Client{Jar: jar, Timeout: 5 * time.Second}
	check := func(client *http.Client, bearer string, want int) {
		t.Helper()
		r, _ := http.NewRequest("GET", public.String()+"/api/auth/check", nil)
		if bearer != "" {
			r.Header.Set("Authorization", "Bearer "+bearer)
		}
		res, err := client.Do(r)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != want {
			t.Fatalf("auth check got %d, want %d", res.StatusCode, want)
		}
	}
	check(client, "", 401)
	res, err := client.Post(public.String()+"/api/login", "application/json", bytes.NewBufferString(`{"username":"fixture","password":"fixture-mac-token"}`))
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, res.Body)
	res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("login: %d", res.StatusCode)
	}
	if len(res.Cookies()) != 1 || res.Cookies()[0].Name != auth.DefaultCookieName || res.Cookies()[0].MaxAge <= 0 {
		t.Fatal("proxy changed or expired the Mac session cookie")
	}
	check(client, "", 200)
	check(client, "", 200) // A later browser refresh still carries the Mac session.
	withoutCookies := &http.Client{Timeout: 5 * time.Second}
	check(withoutCookies, "fixture-mac-token", 200)
	check(withoutCookies, "fixture-pi-token", 401)
}

func TestServiceCookieCannotCollideWithManagementCookie(t *testing.T) {
	if _, err := Start(Config{DisableAuth: true, BackendPort: 12345, CookieName: auth.CookieName()}); err == nil {
		t.Fatal("accepted a service cookie that conflicts with management authentication")
	}
}
