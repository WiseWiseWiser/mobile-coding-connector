package auth

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestCookieNameValidation(t *testing.T) {
	for _, name := range []string{DefaultCookieName, "pi-agent-token", "mac_agent.token", "pi-$%token"} {
		if err := ValidateCookieName(name); err != nil {
			t.Errorf("valid name %q: %v", name, err)
		}
	}
	for _, name := range []string{"", "bad name", "bad;name", "bad=name", "bad\r\nname", "非ASCII", ServiceCookieName, "__Host-token", "__Secure-token", "__Http-token"} {
		if err := ValidateCookieName(name); err == nil {
			t.Errorf("accepted invalid or reserved name %q", name)
		}
	}
}

func configureCookieFixture(t *testing.T, name string) {
	t.Helper()
	previousName, previousFile := CookieName(), getCredentialsFile()
	t.Cleanup(func() {
		if err := SetCookieName(previousName); err != nil {
			t.Error(err)
		}
		SetCredentialsFile(previousFile)
	})
	if err := SetCookieName(name); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "credentials")
	if err := os.WriteFile(path, []byte("fixture-management-token\n"), 0600); err != nil {
		t.Fatal(err)
	}
	SetCredentialsFile(path)
}

func TestCustomCookieLoginAndAuthentication(t *testing.T) {
	for _, name := range []string{DefaultCookieName, "pi-agent-token"} {
		t.Run(name, func(t *testing.T) {
			configureCookieFixture(t, name)
			req := httptest.NewRequest("POST", "/api/login", bytes.NewBufferString(`{"username":"fixture","password":"fixture-management-token"}`))
			w := httptest.NewRecorder()
			handleLogin(w, req)
			if w.Code != 200 {
				t.Fatalf("login: %d", w.Code)
			}
			var session *http.Cookie
			for _, cookie := range w.Result().Cookies() {
				if cookie.Name == name && cookie.MaxAge > 0 {
					session = cookie
				}
			}
			if session == nil || !session.HttpOnly || session.Path != "/" || session.SameSite != http.SameSiteLaxMode {
				t.Fatal("missing session cookie or changed security attributes")
			}
			if name != DefaultCookieName {
				for _, cookie := range w.Result().Cookies() {
					if cookie.Name == DefaultCookieName {
						t.Fatal("custom-name login must not overwrite or expire another instance's default-name session")
					}
				}
			}
			for _, endpoint := range []string{"/api/auth/check", "/api/auth/status", "/api/protected"} {
				for _, mode := range []string{"configured-cookie", "bearer", "anonymous", "old-cookie", "matching-cookie-and-bearer", "mismatching-cookie-and-bearer"} {
					t.Run(endpoint+"/"+mode, func(t *testing.T) {
						r := httptest.NewRequest("GET", endpoint, nil)
						want := 200
						switch mode {
						case "configured-cookie":
							r.AddCookie(session)
						case "bearer":
							r.Header.Set("Authorization", "Bearer fixture-management-token")
						case "anonymous":
							want = 401
						case "old-cookie":
							r.AddCookie(&http.Cookie{Name: DefaultCookieName, Value: "fixture-management-token"})
							if name != DefaultCookieName {
								want = 401
							}
						case "matching-cookie-and-bearer":
							r.AddCookie(session)
							r.Header.Set("Authorization", "Bearer fixture-management-token")
						case "mismatching-cookie-and-bearer":
							r.AddCookie(session)
							r.Header.Set("Authorization", "Bearer different-token")
							want = 401
						}
						response := httptest.NewRecorder()
						switch endpoint {
						case "/api/auth/check":
							handleAuthCheck(response, r)
						case "/api/auth/status":
							handleAuthStatus(response, r)
						default:
							Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }), nil).ServeHTTP(response, r)
						}
						if response.Code != want {
							t.Fatalf("got %d, want %d", response.Code, want)
						}
					})
				}
			}
		})
	}
}

func TestCustomCookieProxyIsolation(t *testing.T) {
	configureCookieFixture(t, "pi-agent-token")
	for _, bearer := range []string{"", "fixture-management-token", "mac-app-token"} {
		r := httptest.NewRequest("GET", "http://public.example/", nil)
		for _, cookie := range []*http.Cookie{
			{Name: "pi-agent-token", Value: "pi-first"},
			{Name: DefaultCookieName, Value: "mac-app-token"},
			{Name: "application-session", Value: "keep"},
			{Name: "pi-agent-token", Value: "pi-duplicate"},
			{Name: DefaultCookieName, Value: "mac-duplicate"},
		} {
			r.AddCookie(cookie)
		}
		if bearer != "" {
			r.Header.Set("Authorization", "Bearer "+bearer)
		}
		StripManagementCredentials(r)
		if _, err := r.Cookie("pi-agent-token"); err == nil {
			t.Fatal("Pi management cookie leaked")
		}
		got := r.Cookies()
		if len(got) != 3 || got[0].Name != DefaultCookieName || got[0].Value != "mac-app-token" || got[1].Name != "application-session" || got[2].Value != "mac-duplicate" {
			t.Fatal("application cookies changed or reordered")
		}
		want := ""
		if bearer == "mac-app-token" {
			want = "Bearer mac-app-token"
		}
		if r.Header.Get("Authorization") != want {
			t.Fatal("bearer isolation changed")
		}
	}
}

func TestInvalidCookieNameDoesNotChangeConfiguration(t *testing.T) {
	configureCookieFixture(t, "pi-agent-token")
	if err := SetCookieName("invalid name"); err == nil {
		t.Fatal("expected validation error")
	}
	if CookieName() != "pi-agent-token" {
		t.Fatal("invalid configuration changed current name")
	}
}
