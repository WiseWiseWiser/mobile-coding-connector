# Local macOS Menu Bar Bearer Token Resolution Doctests

Pure-function and source-contract tests for **local** menu-bar auth: resolve a
Bearer token for `ServerClient` against the local loopback server so
`/api/services`, `/api/terminal/*`, `/api/wrk/*`, … stop returning 401 empty menus.

Go package under `macosapp/localauth` is the doctest surface (temp `DataDir`
fixtures — never real `~/.ai-critic`). Swift `ServerClient` attaches
`Authorization: Bearer <token>` when the resolved token is non-empty.

# DSN (Domain Specific Notion)

**Participants**

- **Resolver (`macosapp/localauth`)** — pure `ResolveLocalServerToken(opts)` that
  reads files under `opts.DataDir` (empty → `~/.ai-critic`) and returns
  `(token, source)` where `source` is `config` | `credentials` | `none`.
- **local-agent-config.json** — same schema as local-agent CLI:
  `{ "default": "<server>", "domains": [ { "server", "token" } ] }`.
- **server-credentials** — plaintext one token per line; first non-empty line
  after trim wins.
- **Local loopback servers** — domain match targets after normalize (trim space
  + trailing `/`): `http://localhost:23712` and `http://127.0.0.1:23712`.
- **Authorization helper** — `AuthorizationHeader(token)` → `Bearer <token>` or
  empty string when token is empty (caller omits header).
- **Local ServerClient (Swift)** — menu-bar HTTP client for loopback APIs; must
  set `Authorization: Bearer …` when a token was resolved.
- **Local app profile (`macosapp/appprofile`)** — local product may advertise
  `UsesAuthToken=true` once local auth is required.
- **Test harness** — writes fixture files into a per-leaf temp `DataDir`, calls
  pure Go helpers, or inspects Swift sources; no network.

**Behaviors**

- **Locked resolution order (fall-through only on read/empty, not on HTTP 401):**
  1. **Config** — in `local-agent-config.json`:
     - Prefer a domain whose server matches a local loopback URL after normalize.
     - Else use the domain matching `default` after normalize.
     - Accept token only if non-empty after trim → `source=config`.
     - Missing file, invalid JSON, unreadable, no usable domain, or empty/whitespace
       token → fall through (do **not** treat as fatal).
  2. **Credentials** — first non-empty trimmed line of `server-credentials`
     → `source=credentials`. Skip blank/whitespace-only lines. Missing/empty file
     → fall through.
  3. **None** — `token=""`, `source=none` (requests unauthenticated).
- **Config wins** when both config and credentials have tokens (no credentials
  read needed for outcome).
- **Local match beats default** when a loopback domain has a non-empty token even
  if `default` points at another domain with a different token.
- **Auth header** — non-empty token → `Bearer <token>`; empty → `""` (no bare
  `Bearer` prefix).
- **ServerClient contract** — local Swift client applies Bearer when token present
  and omits (or leaves empty) Authorization when token is empty.
- **Out of scope** — remote app / `remote-agent-config.json`; 401 retry with
  alternate token; skip-list auth for `/api/wrk`; UX auth-error labels.

## Version

0.0.2

## Decision Tree

```
[local menubar Bearer token]
 |
 +-- resolve/                                    (GROUP)  ResolveLocalServerToken(dataDir)
 |    +-- from-config/                           (GROUP)  config yields non-empty token
 |    |    +-- default-domain-token/             (LEAF)   default domain token → config
 |    |    +-- matching-localhost-domain/        (LEAF)   localhost:23712 domain (not default)
 |    |    +-- matching-127-domain/              (LEAF)   127.0.0.1:23712 domain match
 |    |    +-- prefer-local-match-over-default/  (LEAF)   local domain wins over default domain
 |    |    +-- local-empty-uses-default/         (LEAF)   empty local token → default domain
 |    |    +-- config-wins-over-credentials/     (LEAF)   both present → config token
 |    |
 |    +-- fallthrough-credentials/               (GROUP)  config unusable → credentials
 |    |    +-- config-missing/                   (LEAF)   no config file, creds present
 |    |    +-- config-token-empty/               (LEAF)   whitespace token → credentials
 |    |    +-- config-invalid-json/              (LEAF)   bad JSON → credentials
 |    |    +-- credentials-skips-blank-lines/    (LEAF)   first non-empty creds line
 |    |
 |    +-- none/                                  (GROUP)  no usable token
 |         +-- both-missing/                     (LEAF)   neither file → none
 |         +-- credentials-only-blanks/          (LEAF)   blank creds only → none
 |
 +-- auth/                                       (GROUP)  AuthorizationHeader helper
 |    +-- bearer-token/                          (LEAF)   token → Bearer <token>
 |    +-- empty-token/                           (LEAF)   empty → ""
 |
 +-- client/                                     (GROUP)  Swift source contracts
 |    +-- serverclient-sets-bearer/              (LEAF)   ServerClient sets Authorization Bearer
 |
 +-- profile/                                    (GROUP)  local app profile flags
      +-- local-uses-auth-token/                 (LEAF)   Local().UsesAuthToken == true
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `resolve/from-config/default-domain-token` | Default domain has non-empty token → `source=config` |
| 2 | `resolve/from-config/matching-localhost-domain` | Domain matches `http://localhost:23712` (default points elsewhere) → that token |
| 3 | `resolve/from-config/matching-127-domain` | Domain matches `http://127.0.0.1:23712` → that token |
| 4 | `resolve/from-config/prefer-local-match-over-default` | Local + default both have tokens → local wins |
| 5 | `resolve/from-config/local-empty-uses-default` | Empty local domain token → default domain still `config` |
| 6 | `resolve/from-config/config-wins-over-credentials` | Config and credentials both set → config token, not credentials |
| 7 | `resolve/fallthrough-credentials/config-missing` | Missing config, credentials present → first creds token |
| 8 | `resolve/fallthrough-credentials/config-token-empty` | Config token whitespace-only → credentials |
| 9 | `resolve/fallthrough-credentials/config-invalid-json` | Invalid JSON config → credentials |
| 10 | `resolve/fallthrough-credentials/credentials-skips-blank-lines` | Leading blank lines skipped; first non-empty line |
| 11 | `resolve/none/both-missing` | Both files missing → `token=""`, `source=none` |
| 12 | `resolve/none/credentials-only-blanks` | Config missing, credentials only blanks → `none` |
| 13 | `auth/bearer-token` | `AuthorizationHeader("abc")` → `Bearer abc` |
| 14 | `auth/empty-token` | Empty token → empty header (no bare `Bearer`) |
| 15 | `client/serverclient-sets-bearer` | Local `ServerClient.swift` applies Authorization Bearer |
| 16 | `profile/local-uses-auth-token` | `appprofile.Local().UsesAuthToken` is true |

## Parameter Coverage

| Factor | Leaves |
|--------|--------|
| Config yields token (default domain) | default-domain-token |
| Local loopback match (localhost / 127.0.0.1) | matching-localhost-domain, matching-127-domain |
| Local match preferred over default | prefer-local-match-over-default |
| Empty local → default still config | local-empty-uses-default |
| Config vs credentials precedence | config-wins-over-credentials |
| Config missing / empty token / invalid JSON | fallthrough-credentials/* |
| Credentials blank-line skip | credentials-skips-blank-lines |
| No token available | both-missing, credentials-only-blanks |
| Bearer header format | auth/* |
| Swift ServerClient attaches header | client/serverclient-sets-bearer |
| Local profile UsesAuthToken | profile/local-uses-auth-token |
| Temp DataDir isolation (no real home) | all resolve/* leaves |

## How to Run

```sh
doctest vet ./tests/macos-local-token
doctest test ./tests/macos-local-token/...
```

```go
import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/xhd2015/ai-critic/macosapp/appprofile"
	"github.com/xhd2015/ai-critic/macosapp/localauth"
)

// Request is filled by SETUP chain (root → leaf). Op selects Run branch.
type Request struct {
	Op string // resolve | auth | client | profile

	// resolve: fixture controls. Empty DataDir → Run creates a temp dir.
	DataDir string

	// When ConfigPresent is true, write ConfigJSON to
	// {DataDir}/local-agent-config.json (content may be invalid).
	// When false, omit the file (missing).
	ConfigPresent bool
	ConfigJSON    string

	// When CredentialsPresent is true, write CredentialsText to
	// {DataDir}/server-credentials. When false, omit the file.
	CredentialsPresent bool
	CredentialsText    string

	// auth
	Token string

	// client
	ClientLeaf string

	// profile: "local" | "remote"
	ProfileName string
}

// Response is produced by Run for Assert.
type Response struct {
	// resolve
	Token  string
	Source string // "config" | "credentials" | "none"

	// auth
	AuthHeader string

	// client (source contract)
	SourcesChecked       []string
	SetsAuthorization    bool // true if sources set Authorization header
	UsesBearerScheme     bool // true if Bearer scheme appears with token plumbing
	OmitsBareBearerEmpty bool // true if empty-token path avoids bare "Bearer "

	// profile
	UsesAuthToken  bool
	ConfigFileName string
	SpawnsDaemon   bool
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	switch req.Op {
	case "resolve":
		return runResolve(t, req, resp)
	case "auth":
		resp.AuthHeader = localauth.AuthorizationHeader(req.Token)
		return resp, nil
	case "client":
		return runClientContract(t, req, resp)
	case "profile":
		return runProfile(t, req, resp)
	default:
		return nil, fmt.Errorf("unknown op %q", req.Op)
	}
}

func runResolve(t *testing.T, req *Request, resp *Response) (*Response, error) {
	dataDir := req.DataDir
	if dataDir == "" {
		dir, err := os.MkdirTemp("", "macos-local-token-*")
		if err != nil {
			return nil, err
		}
		t.Cleanup(func() { _ = os.RemoveAll(dir) })
		dataDir = dir
		req.DataDir = dir
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}

	if req.ConfigPresent {
		path := filepath.Join(dataDir, "local-agent-config.json")
		if err := os.WriteFile(path, []byte(req.ConfigJSON), 0o600); err != nil {
			return nil, err
		}
	}
	if req.CredentialsPresent {
		path := filepath.Join(dataDir, "server-credentials")
		if err := os.WriteFile(path, []byte(req.CredentialsText), 0o600); err != nil {
			return nil, err
		}
	}

	token, source := localauth.ResolveLocalServerToken(localauth.Options{
		DataDir: dataDir,
	})
	resp.Token = token
	resp.Source = string(source)
	return resp, nil
}

func runProfile(t *testing.T, req *Request, resp *Response) (*Response, error) {
	var p appprofile.Profile
	switch req.ProfileName {
	case "local":
		p = appprofile.Local()
	case "remote":
		p = appprofile.Remote()
	default:
		return nil, fmt.Errorf("unknown ProfileName %q", req.ProfileName)
	}
	resp.UsesAuthToken = p.UsesAuthToken
	resp.ConfigFileName = p.ConfigFileName
	resp.SpawnsDaemon = p.SpawnsDaemon
	return resp, nil
}

func runClientContract(t *testing.T, req *Request, resp *Response) (*Response, error) {
	moduleRoot, err := findModuleRoot()
	if err != nil {
		return nil, err
	}

	switch req.ClientLeaf {
	case "serverclient-sets-bearer":
		// Inspect local ServerClient (and shared request helpers if any) for
		// Authorization Bearer application on outgoing requests.
		candidates := []string{
			filepath.Join(moduleRoot, "macos-ai-critic", "ai-critic-macos", "ServerClient.swift"),
			filepath.Join(moduleRoot, "macos-ai-critic", "Shared", "ServerClient.swift"),
			filepath.Join(moduleRoot, "macos-ai-critic", "ai-critic-macos", "LocalAuth.swift"),
			filepath.Join(moduleRoot, "macos-ai-critic", "Shared", "LocalAuth.swift"),
			filepath.Join(moduleRoot, "macos-ai-critic", "Shared", "AuthHeader.swift"),
		}
		var combined strings.Builder
		for _, p := range candidates {
			data, err := os.ReadFile(p)
			if err != nil {
				continue
			}
			resp.SourcesChecked = append(resp.SourcesChecked, p)
			combined.Write(data)
			combined.WriteByte('\n')
		}
		src := combined.String()
		if src == "" {
			// RED until ServerClient sources are found/updated.
			return resp, nil
		}

		// Must set Authorization header somewhere in the request path.
		resp.SetsAuthorization = strings.Contains(src, "Authorization") ||
			strings.Contains(src, "authorization")

		// Bearer scheme with token plumbing (not a dead string alone).
		bearerWithToken := regexp.MustCompile(`Bearer\s*\\?\(?\s*\w|Bearer\s+\(|"Bearer\s*"\s*\+|Bearer \$\{|Authorization.*Bearer|Bearer.*token`).MatchString(src) ||
			(strings.Contains(src, "Bearer") && (strings.Contains(src, "token") || strings.Contains(src, "Token") || strings.Contains(src, "authToken") || strings.Contains(src, "AuthToken")))
		resp.UsesBearerScheme = bearerWithToken ||
			(strings.Contains(src, "Bearer") && strings.Contains(src, "Authorization"))

		// Empty-token path should not force a bare "Bearer " always-on header.
		// Accept patterns: guard on non-empty token, or AuthorizationHeader-style helper.
		hasGuard := regexp.MustCompile(`(?i)(if|guard).{0,80}(token|authToken|authorization).{0,80}(isEmpty|!.*isEmpty|isEmpty == false|count > 0)`).MatchString(src) ||
			strings.Contains(src, "AuthorizationHeader") ||
			regexp.MustCompile(`(?s)token\s*(!=|==)\s*"".{0,200}Authorization|!token\.isEmpty.{0,200}Authorization`).MatchString(src)
		alwaysBareBearer := regexp.MustCompile(`setValue\(\s*"Bearer\s*"\s*,\s*forHTTPHeaderField:\s*"Authorization"`).MatchString(src) && !hasGuard
		resp.OmitsBareBearerEmpty = !alwaysBareBearer && (hasGuard || resp.UsesBearerScheme)
		return resp, nil
	default:
		return nil, fmt.Errorf("unknown ClientLeaf %q", req.ClientLeaf)
	}
}

func findModuleRoot() (string, error) {
	// Prefer walking from DOCTEST_ROOT when the harness injects it as a variable.
	if DOCTEST_ROOT != "" {
		for dir := DOCTEST_ROOT; ; dir = filepath.Dir(dir) {
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				return dir, nil
			}
			if filepath.Dir(dir) == dir {
				break
			}
		}
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}
```
