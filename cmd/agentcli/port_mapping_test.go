package agentcli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/xhd2015/ai-critic/client"
)

// portStub is a fake server for the port-mapping endpoints. It records every
// request so a test can assert the order and the bodies, which is where the
// mapping commands do their real work.
type portStub struct {
	mu         sync.Mutex
	calls      []string
	bodies     []string
	forwards   []map[string]any
	names      map[string]string
	auth       map[string]map[string]any
	failMethod string
	failPath   string
	status     int
	bodyErr    string
}

func newPortStub() *portStub {
	return &portStub{names: map[string]string{}, auth: map[string]map[string]any{}}
}

func (s *portStub) record(r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, r.Method+" "+r.URL.RequestURI())
	raw, _ := io.ReadAll(r.Body)
	s.bodies = append(s.bodies, string(raw))
}

func (s *portStub) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.record(r)
		if s.failMethod != "" && r.Method == s.failMethod && r.URL.Path == s.failPath {
			w.WriteHeader(s.status)
			io.WriteString(w, s.bodyErr)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/ports" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(s.forwards)
		case r.URL.Path == "/api/ports" && r.Method == http.MethodPost:
			var body map[string]any
			_ = json.NewDecoder(strings.NewReader(s.lastBody())).Decode(&body)
			port := int(body["port"].(float64))
			host, _ := body["label"].(string)
			provider, _ := body["provider"].(string)
			fwd := map[string]any{
				"localPort": port, "label": host, "provider": provider,
				"status": "active", "publicUrl": "https://" + host,
			}
			if auth, ok := body["auth"].(map[string]any); ok {
				if required, _ := auth["required"].(bool); required {
					if _, has := auth["token"]; !has {
						if mode, _ := auth["tokenMode"].(string); mode != "shared" {
							auth["token"] = "generated-from-stub"
							auth["ephemeral"] = true
							auth["tokenMode"] = "ephemeral"
						}
					}
					fwd["auth"] = auth
				}
			}
			s.forwards = []map[string]any{fwd}
			s.names[fmt.Sprint(port)] = host
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(fwd)
		case r.URL.Path == "/api/ports" && r.Method == http.MethodDelete:
			port := r.URL.Query().Get("port")
			s.forwards = nil
			_ = port
			io.WriteString(w, `{"success":true}`)
		case r.URL.Path == "/api/ports/mapping-names" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(s.names)
		case r.URL.Path == "/api/ports/mapping-names" && r.Method == http.MethodPost:
			var body map[string]any
			_ = json.NewDecoder(strings.NewReader(s.lastBody())).Decode(&body)
			s.names[fmt.Sprint(int(body["port"].(float64)))] = body["domain"].(string)
			w.WriteHeader(http.StatusCreated)
			io.WriteString(w, `{"success":true}`)
		case r.URL.Path == "/api/ports/mapping-names" && r.Method == http.MethodDelete:
			delete(s.names, r.URL.Query().Get("port"))
			io.WriteString(w, `{"success":true}`)
		case r.URL.Path == "/api/ports/mapping-auth" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(s.auth)
		case r.URL.Path == "/api/ports/mapping-auth" && r.Method == http.MethodDelete:
			delete(s.auth, r.URL.Query().Get("port"))
			io.WriteString(w, `{"success":true}`)
		case r.URL.Path == "/api/ports/local":
			io.WriteString(w, `[{"port":9999,"pid":1,"command":"backend"}]`)
		default:
			t := fmt.Sprintf("unhandled %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, t)
		}
	})
}

func (s *portStub) lastBody() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.bodies) == 0 {
		return ""
	}
	return s.bodies[len(s.bodies)-1]
}

func (s *portStub) callsCopy() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.calls...)
}

func (s *portStub) bodiesCopy() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.bodies...)
}

func runMapping(t *testing.T, stub *portStub, args ...string) (string, error) {
	t.Helper()
	server := httptest.NewServer(stub.handler())
	defer server.Close()
	var out bytes.Buffer
	var runErr error
	withStdout(&out, func() {
		runErr = runPortMapping(func() (*client.Client, error) {
			return client.New(server.URL, ""), nil
		}, args)
	})
	return out.String(), runErr
}

func TestPortMappingAddSavesNameThenPublishes(t *testing.T) {
	stub := newPortStub()
	out, err := runMapping(t, stub, "add", "9999", "--hostname", "house-design.xhd2015.xyz")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	// The name must be remembered before the forward, mirroring the web UI, so
	// the hostname survives a later delete.
	want := []string{
		"POST /api/ports/mapping-names",
		"POST /api/ports",
		"GET /api/ports",
		"GET /api/ports/local",
	}
	if got := stub.callsCopy(); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("calls = %v, want %v", got, want)
	}
	bodies := stub.bodiesCopy()
	if !strings.Contains(bodies[0], `"domain":"house-design.xhd2015.xyz"`) {
		t.Fatalf("mapping-names body = %s", bodies[0])
	}
	for _, want := range []string{`"port":9999`, `"label":"house-design.xhd2015.xyz"`, `"provider":"cloudflare_owned"`} {
		if !strings.Contains(bodies[1], want) {
			t.Fatalf("POST /api/ports body = %s, want %s", bodies[1], want)
		}
	}
	if !strings.Contains(out, "https://house-design.xhd2015.xyz") {
		t.Fatalf("output missing public url: %s", out)
	}
}

func TestPortMappingProviderAliases(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"", "cloudflare_owned"},
		{"owned", "cloudflare_owned"},
		{"quick", "cloudflare_quick"},
		{"cloudflare_quick", "cloudflare_quick"},
		{"localtunnel", "localtunnel"},
	} {
		stub := newPortStub()
		args := []string{"add", "9999", "--hostname", "h.example.com"}
		if tc.in != "" {
			args = append(args, "--provider", tc.in)
		}
		if _, err := runMapping(t, stub, args...); err != nil {
			t.Fatalf("provider %q: %v", tc.in, err)
		}
		var body map[string]any
		for _, b := range stub.bodiesCopy() {
			if strings.Contains(b, `"label"`) {
				_ = json.Unmarshal([]byte(b), &body)
			}
		}
		if body["provider"] != tc.want {
			t.Fatalf("provider %q sent %v, want %q", tc.in, body["provider"], tc.want)
		}
	}
}

func TestPortMappingAddRejectsUnknownProvider(t *testing.T) {
	stub := newPortStub()
	_, err := runMapping(t, stub, "add", "9999", "--hostname", "h.example.com", "--provider", "aws")
	if err == nil || !strings.Contains(err.Error(), "unknown provider") {
		t.Fatalf("err = %v", err)
	}
	if len(stub.callsCopy()) != 0 {
		t.Fatalf("no request should be sent, got %v", stub.callsCopy())
	}
}

func TestPortMappingAddPortAlreadyForwardedPointsAtUpdate(t *testing.T) {
	stub := newPortStub()
	stub.failMethod = http.MethodPost
	stub.failPath = "/api/ports"
	stub.status = http.StatusBadRequest
	stub.bodyErr = "port 9999 is already being forwarded"
	_, err := runMapping(t, stub, "add", "9999", "--hostname", "h.example.com")
	if err == nil || !strings.Contains(err.Error(), "port mapping update 9999") {
		t.Fatalf("err = %v, want it to point at update", err)
	}
}

func TestPortMappingUpdateRemovesThenAdds(t *testing.T) {
	stub := newPortStub()
	stub.forwards = []map[string]any{{
		"localPort": 9999, "label": "old.xhd2015.xyz",
		"provider": "cloudflare_owned", "status": "active",
		"publicUrl": "https://old.xhd2015.xyz",
	}}
	out, err := runMapping(t, stub, "update", "9999", "--hostname", "new.xhd2015.xyz")
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	// The old mapping has to go before the new one: the server rejects a second
	// mapping on the same port.
	calls := stub.callsCopy()
	delIdx, postIdx := -1, -1
	for i, c := range calls {
		if c == "DELETE /api/ports?port=9999" && delIdx < 0 {
			delIdx = i
		}
		if c == "POST /api/ports" && postIdx < 0 {
			postIdx = i
		}
	}
	if delIdx < 0 || postIdx < 0 || delIdx > postIdx {
		t.Fatalf("expected DELETE before POST, got %v", calls)
	}
	if !strings.Contains(out, "was old.xhd2015.xyz") {
		t.Fatalf("output should name the previous hostname: %s", out)
	}
}

func TestPortMappingUpdateRestoresOldMappingOnFailure(t *testing.T) {
	stub := newPortStub()
	stub.forwards = []map[string]any{{
		"localPort": 9999, "label": "old.xhd2015.xyz",
		"provider": "cloudflare_owned", "status": "active",
	}}
	stub.failMethod = http.MethodPost
	stub.failPath = "/api/ports"
	stub.status = http.StatusInternalServerError
	stub.bodyErr = "tunnel boom"
	_, err := runMapping(t, stub, "update", "9999", "--hostname", "new.xhd2015.xyz")
	if err == nil {
		t.Fatal("update should fail")
	}
	// A failed update must not leave the port unpublished.
	var reAdded bool
	for _, b := range stub.bodiesCopy() {
		if strings.Contains(b, `"label":"old.xhd2015.xyz"`) {
			reAdded = true
		}
	}
	if !reAdded {
		t.Fatalf("old mapping was not restored; bodies = %v", stub.bodiesCopy())
	}
}

func TestPortMappingUpdateRequiresAnExistingMapping(t *testing.T) {
	stub := newPortStub()
	_, err := runMapping(t, stub, "update", "9999", "--hostname", "new.xhd2015.xyz")
	if err == nil || !strings.Contains(err.Error(), "is not mapped") {
		t.Fatalf("err = %v", err)
	}
}

func TestPortMappingDeleteKeepsSavedName(t *testing.T) {
	stub := newPortStub()
	stub.forwards = []map[string]any{{"localPort": 9999, "label": "h.xhd2015.xyz", "status": "active"}}
	stub.names["9999"] = "h.xhd2015.xyz"
	out, err := runMapping(t, stub, "delete", "9999")
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	for _, c := range stub.callsCopy() {
		if strings.HasPrefix(c, "DELETE /api/ports/mapping-names") {
			t.Fatalf("delete must keep the saved name, calls = %v", stub.callsCopy())
		}
	}
	if !strings.Contains(out, "kept") {
		t.Fatalf("output should say the name was kept: %s", out)
	}
}

func TestPortMappingDeleteForgetClearsSavedName(t *testing.T) {
	stub := newPortStub()
	stub.forwards = []map[string]any{{"localPort": 9999, "label": "h.xhd2015.xyz", "status": "active"}}
	stub.names["9999"] = "h.xhd2015.xyz"
	out, err := runMapping(t, stub, "delete", "9999", "--forget")
	if err != nil {
		t.Fatalf("delete --forget: %v", err)
	}
	var forgot bool
	for _, c := range stub.callsCopy() {
		if c == "DELETE /api/ports/mapping-names?port=9999" {
			forgot = true
		}
	}
	if !forgot {
		t.Fatalf("--forget should clear the name, calls = %v", stub.callsCopy())
	}
	if !strings.Contains(out, "cleared") {
		t.Fatalf("output should say the name was cleared: %s", out)
	}
}

// A saved name with no live mapping is exactly the state 3080/9393 are in, and
// it is what makes re-adding a port convenient — so list must surface it.
func TestPortMappingListShowsOrphanSavedNames(t *testing.T) {
	stub := newPortStub()
	stub.names["3080"] = "dsh-on-mac.xhd2015.xyz"
	stub.names["9999"] = "house-design.xhd2015.xyz"
	stub.forwards = []map[string]any{{
		"localPort": 9999, "label": "house-design.xhd2015.xyz",
		"provider": "cloudflare_owned", "status": "active",
		"publicUrl": "https://house-design.xhd2015.xyz",
	}}
	out, err := runMapping(t, stub, "list")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out, "3080") || !strings.Contains(out, "inactive") {
		t.Fatalf("list should show the orphan saved name:\n%s", out)
	}
	if !strings.Contains(out, "https://house-design.xhd2015.xyz") {
		t.Fatalf("list should show the active mapping:\n%s", out)
	}
}

func TestPortMappingListJSON(t *testing.T) {
	stub := newPortStub()
	stub.names["3080"] = "dsh-on-mac.xhd2015.xyz"
	out, err := runMapping(t, stub, "list", "--json")
	if err != nil {
		t.Fatalf("list --json: %v", err)
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, out)
	}
	if len(rows) != 1 || rows[0]["port"] != float64(3080) || rows[0]["savedName"] != "dsh-on-mac.xhd2015.xyz" {
		t.Fatalf("rows = %#v", rows)
	}
}

// The SOURCE column exists because two mappings with different origins used to render as
// identical rows: 'service list' showed the service's forward, 'port mapping list' showed the
// same URL with nothing saying where it came from.
func TestPortForwardSourceNamesTheOrigin(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   client.PortForwardInfo
		want string
	}{
		// A service's forward and a hand-added one are both this type — which is exactly why the
		// column may not claim "service" or "manual".
		{"service forward", client.PortForwardInfo{Type: "port_forward"}, "forward"},
		{"hand-added forward", client.PortForwardInfo{Type: "port_forward"}, "forward"},
		{"no type reported", client.PortForwardInfo{}, "forward"},
		{"boot domain tunnel", client.PortForwardInfo{Type: "domain", Bootstrap: true}, "domain(boot)"},
		{"requested domain tunnel", client.PortForwardInfo{Type: "domain"}, "domain"},
		{"exposed url", client.PortForwardInfo{Type: "exposed_url"}, "exposed"},
		// An unknown future type is shown as-is rather than dropped or mislabelled.
		{"unknown type", client.PortForwardInfo{Type: "something_new"}, "something_new"},
	} {
		if got := portForwardSource(tc.in); got != tc.want {
			t.Errorf("%s: portForwardSource = %q, want %q", tc.name, got, tc.want)
		}
	}
}

// The two live rows that caused the confusion must not collapse to the same label.
//
// This asserts the labels by value, not merely that they differ: an earlier version checked only
// inequality and still passed when Bootstrap was ignored — "domain" and "forward" are different
// strings, so the boot tunnel was indistinguishable from a requested one and nothing caught it.
func TestPortForwardSourceDistinguishesTheReportedPair(t *testing.T) {
	cases := []struct {
		name string
		in   client.PortForwardInfo
		want string
	}{
		{"8095 (the service's live row)", client.PortForwardInfo{LocalPort: 8095, Type: "port_forward"}, "forward"},
		{"23712 (the boot domain tunnel)", client.PortForwardInfo{LocalPort: 23712, Type: "domain", Bootstrap: true}, "domain(boot)"},
	}
	seen := map[string]string{}
	for _, tc := range cases {
		got := portForwardSource(tc.in)
		if got != tc.want {
			t.Errorf("%s: portForwardSource = %q, want %q", tc.name, got, tc.want)
		}
		seen[got] = tc.name
	}
	if len(seen) != len(cases) {
		t.Fatalf("two rows share a label (%v); the confusion would remain", seen)
	}
}

// source is in the machine-readable output too, or scripts keep the ambiguity.
func TestPortMappingListJSONCarriesSource(t *testing.T) {
	stub := newPortStub()
	out, err := runMapping(t, stub, "list", "--json")
	if err != nil {
		t.Fatalf("list --json: %v", err)
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, out)
	}
	for _, r := range rows {
		if r["source"] == nil || r["source"] == "" {
			t.Fatalf("row %#v has no source", r)
		}
	}
}

func TestPortMappingHelpUsesProfileName(t *testing.T) {
	for _, tc := range []struct {
		profile Profile
		want    string
	}{
		{LocalProfile(), "Usage: local-agent port mapping"},
		{RemoteProfile(), "Usage: remote-agent port mapping"},
	} {
		var out bytes.Buffer
		active = tc.profile
		if err := RunWithWriters(tc.profile, []string{"port", "mapping", "-h"}, &out, &out); err != nil {
			t.Fatalf("%s: %v", tc.profile.Name, err)
		}
		if !strings.Contains(out.String(), tc.want) {
			t.Fatalf("%s help missing %q:\n%s", tc.profile.Name, tc.want, out.String())
		}
	}
}

func TestPortMappingAddRequiresHostnameAndPort(t *testing.T) {
	stub := newPortStub()
	if _, err := runMapping(t, stub, "add", "9999"); err == nil || !strings.Contains(err.Error(), "--hostname") {
		t.Fatalf("err = %v", err)
	}
	if _, err := runMapping(t, stub, "add", "--hostname", "h.example.com"); err == nil || !strings.Contains(err.Error(), "requires a port") {
		t.Fatalf("err = %v", err)
	}
	if _, err := runMapping(t, stub, "add", "abc", "--hostname", "h.example.com"); err == nil || !strings.Contains(err.Error(), "invalid port") {
		t.Fatalf("err = %v", err)
	}
}

// The port name is in the URL of every route in this area, so it must not be
// possible for the top-level help to drift back to a hardcoded binary name.
func TestPortHelpUsesProfileName(t *testing.T) {
	for _, tc := range []struct {
		profile Profile
		want    string
	}{
		{LocalProfile(), "Usage: local-agent port <subcommand>"},
		{RemoteProfile(), "Usage: remote-agent port <subcommand>"},
	} {
		var out bytes.Buffer
		active = tc.profile
		if err := RunWithWriters(tc.profile, []string{"port", "-h"}, &out, &out); err != nil {
			t.Fatalf("%s: %v", tc.profile.Name, err)
		}
		if !strings.Contains(out.String(), tc.want) {
			t.Fatalf("%s help missing %q:\n%s", tc.profile.Name, tc.want, out.String())
		}
	}
}

func TestPortMappingAddRequireAuthSendsFlag(t *testing.T) {
	stub := newPortStub()
	out, err := runMapping(t, stub, "add", "8422", "--hostname", "eluc-on-mac.xhd2015.xyz", "--require-auth")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	var body map[string]any
	for _, b := range stub.bodiesCopy() {
		if strings.Contains(b, `"label":"eluc-on-mac.xhd2015.xyz"`) {
			_ = json.Unmarshal([]byte(b), &body)
		}
	}
	auth, _ := body["auth"].(map[string]any)
	if auth["required"] != true {
		t.Fatalf("POST /api/ports body = %v, want auth.required true", body)
	}
	if auth["tokenMode"] == "shared" {
		t.Fatalf("default require-auth must not send shared: %v", auth)
	}
	if !strings.Contains(out, "Auth:") || !strings.Contains(out, "required") {
		t.Fatalf("output missing auth: %s", out)
	}
	if !strings.Contains(out, "Token:") || !strings.Contains(out, "generated-from-stub") {
		t.Fatalf("output missing generated token: %s", out)
	}
}

func TestPortMappingAddAuthFlagsRequireRequireAuth(t *testing.T) {
	stub := newPortStub()
	_, err := runMapping(t, stub, "add", "8422", "--hostname", "eluc-on-mac.xhd2015.xyz", "--auth-token", "s3cret")
	if err == nil || !strings.Contains(err.Error(), "--require-auth") {
		t.Fatalf("err = %v, want --require-auth", err)
	}
	if len(stub.callsCopy()) != 0 {
		t.Fatalf("no request should be sent, got %v", stub.callsCopy())
	}
}

func TestPortMappingListShowsAuthColumn(t *testing.T) {
	stub := newPortStub()
	stub.forwards = []map[string]any{{
		"localPort": 8422, "label": "eluc-on-mac.xhd2015.xyz",
		"provider": "cloudflare_owned", "status": "active",
		"publicUrl": "https://eluc-on-mac.xhd2015.xyz",
		"auth":      map[string]any{"required": true, "ephemeral": true, "tokenMode": "ephemeral"},
	}}
	stub.names["3080"] = "dsh-on-mac.xhd2015.xyz"
	stub.auth["8422"] = map[string]any{"required": true, "ephemeral": true, "tokenMode": "ephemeral", "token": "map-token"}
	stub.auth["3080"] = map[string]any{"required": true, "ephemeral": true, "tokenMode": "ephemeral", "token": "inactive-token"}
	out, err := runMapping(t, stub, "list")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out, "AUTH") || !strings.Contains(out, "required") {
		t.Fatalf("list missing AUTH column:\n%s", out)
	}
	if strings.Contains(out, "map-token") || strings.Contains(out, "inactive-token") {
		t.Fatalf("default list leaked tokens:\n%s", out)
	}
}

func TestPortMappingListShowToken(t *testing.T) {
	stub := newPortStub()
	stub.forwards = []map[string]any{{
		"localPort": 8422, "label": "eluc-on-mac.xhd2015.xyz",
		"provider": "cloudflare_owned", "status": "active",
		"publicUrl": "https://eluc-on-mac.xhd2015.xyz",
		"auth":      map[string]any{"required": true, "ephemeral": true, "tokenMode": "ephemeral"},
	}}
	stub.auth["8422"] = map[string]any{"required": true, "ephemeral": true, "tokenMode": "ephemeral", "token": "map-token"}
	out, err := runMapping(t, stub, "list", "--show-token")
	if err != nil {
		t.Fatalf("list --show-token: %v", err)
	}
	if !strings.Contains(out, "map-token") || !strings.Contains(out, "ephemeral") {
		t.Fatalf("missing mapping token:\n%s", out)
	}
}

func TestPortMappingListJSONOmitsTokenByDefault(t *testing.T) {
	stub := newPortStub()
	stub.forwards = []map[string]any{{
		"localPort": 8422, "status": "active", "provider": "cloudflare_owned",
		"auth": map[string]any{"required": true, "tokenMode": "ephemeral"},
	}}
	stub.auth["8422"] = map[string]any{"required": true, "tokenMode": "ephemeral", "token": "secret"}
	out, err := runMapping(t, stub, "list", "--json")
	if err != nil {
		t.Fatalf("list --json: %v", err)
	}
	if strings.Contains(out, "secret") {
		t.Fatalf("json leaked token: %s", out)
	}
}

func TestPortMappingSharedTokenConflictsWithAuthToken(t *testing.T) {
	stub := newPortStub()
	_, err := runMapping(t, stub, "add", "8422", "--hostname", "eluc-on-mac.xhd2015.xyz",
		"--require-auth", "--shared-token", "--auth-token", "x")
	if err == nil || !strings.Contains(err.Error(), "--shared-token") {
		t.Fatalf("err = %v", err)
	}
}

func TestPortMappingDeleteForgetClearsAuth(t *testing.T) {
	stub := newPortStub()
	stub.forwards = []map[string]any{{"localPort": 8422, "label": "eluc-on-mac.xhd2015.xyz", "status": "active"}}
	stub.names["8422"] = "eluc-on-mac.xhd2015.xyz"
	stub.auth["8422"] = map[string]any{"required": true}
	if _, err := runMapping(t, stub, "delete", "8422", "--forget"); err != nil {
		t.Fatalf("delete --forget: %v", err)
	}
	var forgotAuth bool
	for _, c := range stub.callsCopy() {
		if c == "DELETE /api/ports/mapping-auth?port=8422" {
			forgotAuth = true
		}
	}
	if !forgotAuth {
		t.Fatalf("--forget should clear auth flags, calls = %v", stub.callsCopy())
	}
}
