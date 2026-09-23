package authproxy

import (
	_ "embed"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

const (
	loginPath   = "/_auth/login"
	expiredPath = "/_auth/expired"
)

//go:embed login.html
var loginHTML string

//go:embed expired.html
var expiredHTML string

type handler struct {
	cfg   Config
	proxy *httputil.ReverseProxy
}

func newHandler(cfg Config, backend *url.URL) (http.Handler, error) {
	proxy := httputil.NewSingleHostReverseProxy(backend)
	proxy.FlushInterval = -1
	mux := http.NewServeMux()
	h := &handler{cfg: cfg, proxy: proxy}
	mux.HandleFunc(loginPath, h.handleLogin)
	mux.HandleFunc(expiredPath, h.handleExpired)
	mux.HandleFunc("/", h.handleProxy)
	return mux, nil
}

func (h *handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		serveHTML(w, http.StatusOK, loginHTML)
		return
	case http.MethodPost:
		username, token, err := readLoginCredentials(r)
		if err != nil {
			h.loginFailure(w, r, "Username and token are required")
			return
		}
		if !h.cfg.userOK(username) || !h.cfg.tokenOK(token) {
			h.loginFailure(w, r, "Invalid username or token")
			return
		}
		if err := h.setSessionCookie(w, r, username, token); err != nil {
			h.loginFailure(w, r, "Failed to create session")
			return
		}
		if wantsJSON(r) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			return
		}
		http.Redirect(w, r, safeNextPath(r.URL.Query().Get("next")), http.StatusFound)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *handler) handleExpired(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	serveHTML(w, http.StatusOK, expiredHTML)
}

func (h *handler) handleProxy(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/_auth/") {
		http.NotFound(w, r)
		return
	}

	state, sess := h.authState(r)
	switch state {
	case authMissing, authExpired:
		h.reject(w, r, authMissing)
		return
	case authWrong:
		h.reject(w, r, authWrong)
		return
	}

	if sess != nil {
		_ = h.refreshCookie(w, r, *sess)
	}

	stripAuthCookie(r, h.cfg.CookieName)
	h.proxy.ServeHTTP(w, r)
}

type authState int

const (
	authOK authState = iota
	authMissing
	authExpired
	authWrong
)

func (h *handler) authState(r *http.Request) (authState, *session) {
	if token := bearerToken(r); token != "" {
		if h.cfg.tokenOK(token) {
			return authOK, nil
		}
		return authWrong, nil
	}

	cookie, err := r.Cookie(h.cfg.CookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return authMissing, nil
	}

	sess, err := decodeSession(h.cfg.Secret, cookie.Value)
	if err != nil {
		return authWrong, nil
	}
	now := h.cfg.Now()
	if sess.expiredAt(now) {
		return authExpired, nil
	}
	if h.cfg.AuthUser != "" && sess.User != h.cfg.AuthUser {
		return authWrong, nil
	}
	if !h.cfg.fingerprintOK(sess.TokenFP) {
		return authWrong, nil
	}
	return authOK, sess
}

func (h *handler) reject(w http.ResponseWriter, r *http.Request, state authState) {
	if isWebSocketUpgrade(r) || !wantsHTML(r) {
		msg := "unauthorized"
		if state == authWrong {
			msg = "login expired"
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
		return
	}
	if state == authWrong {
		http.Redirect(w, r, expiredPath, http.StatusFound)
		return
	}
	next := safeNextPath(r.URL.RequestURI())
	http.Redirect(w, r, loginPath+"?next="+url.QueryEscape(next), http.StatusFound)
}

func (h *handler) loginFailure(w http.ResponseWriter, r *http.Request, message string) {
	if wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
		return
	}
	serveHTML(w, http.StatusUnauthorized, loginHTML)
}

func (h *handler) setSessionCookie(w http.ResponseWriter, r *http.Request, user, token string) error {
	now := h.cfg.Now()
	fp := ""
	if token != "" {
		fp = tokenFingerprint(token)
	}
	sess := session{
		User:    user,
		TokenFP: fp,
		Expires: now.Add(h.cfg.TTL).Unix(),
	}
	return h.refreshCookie(w, r, sess)
}

func (h *handler) refreshCookie(w http.ResponseWriter, r *http.Request, sess session) error {
	if sess.TokenFP == "" {
		return nil
	}
	now := h.cfg.Now()
	sess.Expires = now.Add(h.cfg.TTL).Unix()
	value, err := encodeSession(h.cfg.Secret, sess)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.CookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   isHTTPS(r),
		Expires:  time.Unix(sess.Expires, 0),
	})
	return nil
}

func readLoginCredentials(r *http.Request) (string, string, error) {
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		var req struct {
			Username string `json:"username"`
			Token    string `json:"token"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
			return "", "", err
		}
		token := strings.TrimSpace(req.Token)
		if token == "" {
			token = strings.TrimSpace(req.Password)
		}
		user := strings.TrimSpace(req.Username)
		if user == "" || token == "" {
			return "", "", io.ErrUnexpectedEOF
		}
		return user, token, nil
	}
	if err := r.ParseForm(); err != nil {
		return "", "", err
	}
	user := strings.TrimSpace(r.Form.Get("username"))
	token := strings.TrimSpace(r.Form.Get("token"))
	if token == "" {
		token = strings.TrimSpace(r.Form.Get("password"))
	}
	if user == "" || token == "" {
		return "", "", io.ErrUnexpectedEOF
	}
	return user, token, nil
}

func serveHTML(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body)
}

func wantsJSON(r *http.Request) bool {
	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	if strings.Contains(contentType, "application/json") {
		return true
	}
	accept := strings.ToLower(r.Header.Get("Accept"))
	return strings.Contains(accept, "application/json") && !strings.Contains(accept, "text/html")
}

func wantsHTML(r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	accept := r.Header.Get("Accept")
	if accept == "" {
		return true
	}
	return strings.Contains(accept, "text/html")
}

func isWebSocketUpgrade(r *http.Request) bool {
	if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return true
	}
	if strings.TrimSpace(r.Header.Get("Sec-WebSocket-Key")) != "" {
		return true
	}
	conn := strings.ToLower(r.Header.Get("Connection"))
	return strings.Contains(conn, "upgrade") && strings.Contains(strings.ToLower(r.Header.Get("Upgrade")), "websocket")
}

func isHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func bearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if len(header) < 7 || !strings.EqualFold(header[:7], "bearer ") {
		return ""
	}
	return strings.TrimSpace(header[7:])
}

func safeNextPath(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "/"
	}
	if !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") {
		return "/"
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.IsAbs() || parsed.Host != "" {
		return "/"
	}
	return raw
}

func stripAuthCookie(r *http.Request, cookieName string) {
	cookies := r.Cookies()
	r.Header.Del("Cookie")
	for _, cookie := range cookies {
		if cookie.Name == cookieName {
			continue
		}
		r.AddCookie(cookie)
	}
}
