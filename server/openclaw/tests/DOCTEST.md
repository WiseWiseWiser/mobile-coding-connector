# OpenClaw Mock Integration Doctests

Package-level tests for `server/openclaw` covering config lifecycle, start
validation, mock gateway lifecycle, generated `openclaw.json` rendering,
doctor checks, and HTTP API error mapping.

# DSN (Domain Specific Notion)

The openclaw doctest harness models ai-critic's mocked OpenClaw gateway
integration with Slack Socket Mode.

**Participants**

- **Config store** — `.ai-critic/openclaw.json`; plaintext Slack tokens on disk,
  masked on API GET/PUT responses.
- **Manager** — mock gateway lifecycle; no real `openclaw gateway` subprocess.
- **Renderer** — writes `.ai-critic/openclaw/openclaw.json` with env SecretRefs.
- **Runtime state** — `.ai-critic/openclaw/state.json` tracks `running`, `mock_pid`,
  `mocked`, `started_at`.
- **HTTP API** — `/api/openclaw/{status,start,stop,config,doctor}` via `httptest`.
- **Doctor** — server-side health checks (node, openclaw CLI, slack tokens, mock
  gateway, generated config).

**Behaviors**

- Missing config file yields defaults (`gateway_port=18789`, slack disabled).
- `ValidateStartConfig` gates start: slack enabled requires bot+app tokens and
  socket mode only.
- `Start` sets `running=true`, `mock_pid=4242`, `mocked=true`, writes generated config.
- Second start returns `ALREADY_RUNNING` (HTTP 409); validation errors return 400.
- `Stop` clears running state; idempotent when already stopped.
- `MaskConfig` replaces non-empty tokens with `***` on API responses only.
- `MergeConfig` preserves omitted secrets on partial PUT.
- `Doctor` always warns that integration is mocked; reports slack/gateway/runtime deps.

## Version

0.0.2

## Decision Tree

```
[openclaw mock integration]
 |
 +-- config-lifecycle/                         (grouping)
 |    +-- defaults-missing-file/               (LEAF)  port 18789, slack off
 |    +-- round-trip-preserves-secrets/        (LEAF)  disk plaintext tokens
 |    +-- get-masks-tokens/                    (LEAF)  GET *** masking
 |    +-- put-partial-preserves-secrets/       (LEAF)  PUT omits tokens
 |    +-- put-partial-without-slack-block/     (LEAF)  PUT no slack key
 |    +-- slack-enabled-defaults/              (LEAF)  socket/pairing/mention
 |
 +-- start-validation/                         (grouping)
 |    +-- slack-disabled-ok/                   (LEAF)  start without tokens
 |    +-- slack-enabled-missing-bot/           (LEAF)  400 BAD_REQUEST
 |    +-- slack-enabled-missing-app/           (LEAF)  400 BAD_REQUEST
 |    +-- slack-mode-http-rejected/            (LEAF)  400 unsupported mode
 |
 +-- gateway-lifecycle/                        (grouping)
 |    +-- start-mock-running/                  (LEAF)  running/mock_pid/config
 |    +-- start-already-running-conflict/      (LEAF)  409 CONFLICT
 |    +-- stop-clears-running/                 (LEAF)  running=false
 |    +-- stop-idempotent/                     (LEAF)  stop when stopped
 |    +-- status-reflects-slack/               (LEAF)  slack_enabled/mode
 |    +-- dry-run-valid/                       (LEAF)  checks, mocked=true
 |    +-- dry-run-with-validation-issues/      (LEAF)  issues on bad config
 |
 +-- generated-config/                         (grouping)
 |    +-- gateway-workspace-model/             (LEAF)  port/workspace/model
 |    +-- slack-socket-secret-refs/            (LEAF)  SLACK_* env refs
 |    +-- require-mention-groups/              (LEAF)  groups.requireMention
 |    +-- dm-policy-allow-from/                (LEAF)  dmPolicy/allowFrom
 |
 +-- doctor/                                   (grouping)
      +-- mocked-integration-warn/             (LEAF)  mock_mode warn
      +-- runtime-deps-checks/                 (LEAF)  node + openclaw_cli
      +-- slack-disabled-skip/                 (LEAF)  slack_enabled skip
      +-- slack-tokens-configured/             (LEAF)  tokens ok, plugin warn
      +-- slack-tokens-missing/                (LEAF)  tokens fail
      +-- gateway-running-ok/                  (LEAF)  gateway_running ok
      +-- gateway-not-running-warn/            (LEAF)  gateway warn + hint
      +-- generated-config-present/            (LEAF)  generated_config ok
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `config-lifecycle/defaults-missing-file` | Defaults when config file absent |
| 2 | `config-lifecycle/round-trip-preserves-secrets` | Save/load keeps plaintext tokens on disk |
| 3 | `config-lifecycle/get-masks-tokens` | GET /config masks bot_token and app_token |
| 4 | `config-lifecycle/put-partial-preserves-secrets` | PUT partial merge keeps omitted secrets |
| 5 | `config-lifecycle/put-partial-without-slack-block` | PUT without slack block preserves slack config |
| 6 | `config-lifecycle/slack-enabled-defaults` | Enabling slack applies socket/pairing/mention defaults |
| 7 | `start-validation/slack-disabled-ok` | Slack disabled allows start without tokens |
| 8 | `start-validation/slack-enabled-missing-bot` | Missing bot token → 400 BAD_REQUEST |
| 9 | `start-validation/slack-enabled-missing-app` | Missing app token → 400 BAD_REQUEST |
| 10 | `start-validation/slack-mode-http-rejected` | HTTP mode → 400 BAD_REQUEST |
| 11 | `gateway-lifecycle/start-mock-running` | Start sets mock state and writes generated config |
| 12 | `gateway-lifecycle/start-already-running-conflict` | Second start → 409 CONFLICT |
| 13 | `gateway-lifecycle/stop-clears-running` | Stop clears running flag |
| 14 | `gateway-lifecycle/stop-idempotent` | Stop when already stopped succeeds |
| 15 | `gateway-lifecycle/status-reflects-slack` | Status exposes slack_enabled and slack_mode |
| 16 | `gateway-lifecycle/dry-run-valid` | Dry-run reports mocked integration checks |
| 17 | `gateway-lifecycle/dry-run-with-validation-issues` | Dry-run surfaces validation issues |
| 18 | `generated-config/gateway-workspace-model` | Rendered gateway port, workspace, model |
| 19 | `generated-config/slack-socket-secret-refs` | Slack tokens as env SecretRefs |
| 20 | `generated-config/require-mention-groups` | require_mention adds groups wildcard |
| 21 | `generated-config/dm-policy-allow-from` | dmPolicy and allowFrom propagated |
| 22 | `doctor/mocked-integration-warn` | Doctor always warns integration is mocked |
| 23 | `doctor/runtime-deps-checks` | node and openclaw_cli checks with hints on fail |
| 24 | `doctor/slack-disabled-skip` | Slack disabled → slack_enabled skip |
| 25 | `doctor/slack-tokens-configured` | Tokens ok; plugin and socket mocked (warn) |
| 26 | `doctor/slack-tokens-missing` | Enabled slack without tokens → fail |
| 27 | `doctor/gateway-running-ok` | Running mock gateway → gateway_running ok |
| 28 | `doctor/gateway-not-running-warn` | Stopped gateway → warn with start hint |
| 29 | `doctor/generated-config-present` | After start, generated openclaw.json check ok |

## How to Run

```sh
doctest vet ./server/openclaw/tests
doctest test ./server/openclaw/tests/...
go test ./server/openclaw/... -count=1
go run ./script/build
```

```go
import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/xhd2015/ai-critic/server/openclaw"
)

const (
	OpLoadDefaults      = "load-defaults"
	OpRoundTrip         = "round-trip"
	OpAPIGetConfig      = "api-get-config"
	OpAPIPutConfig      = "api-put-config"
	OpValidate          = "validate"
	OpStart             = "start"
	OpAPIStart          = "api-start"
	OpStop              = "stop"
	OpStatus            = "status"
	OpDryRun            = "dry-run"
	OpAPIDryRun         = "api-dry-run"
	OpRender            = "render"
	OpDoctor            = "doctor"
)

type Request struct {
	Op string

	WriteInitialConfig bool
	GatewayPort        int
	Workspace          string
	Model              string
	SlackEnabled       bool
	SlackMode          string
	BotToken           string
	AppToken           string
	DMPolicy           string
	AllowFrom          []string
	RequireMention     *bool

	PutBody string

	PreStart    bool
	SecondStart bool
}

type Response struct {
	DataDir string

	Config         *openclaw.Config
	ConfigOnDisk   *openclaw.Config
	State          *openclaw.RuntimeState
	Status         *openclaw.Status
	DryRun         *openclaw.DryRunResult
	Doctor         *openclaw.DoctorReport
	ValidationErr  error
	StartErr       error
	SecondStartErr error

	APIStatusCode int
	APIBody       string

	RenderedJSON map[string]any
	RenderedRaw  []byte
	GeneratedPath string
}

func Run(t *testing.T, req *Request) (*Response, error) {
	dir := t.TempDir()
	openclaw.SetTestDataDir(dir)
	t.Cleanup(func() { openclaw.SetTestDataDir("") })

	resp := &Response{DataDir: dir}
	m := openclaw.GetManager()

	if req.WriteInitialConfig {
		cfg := buildConfig(req)
		if err := openclaw.SaveConfig(cfg); err != nil {
			return nil, err
		}
	}

	if req.PreStart {
		if err := m.Start(); err != nil {
			return nil, err
		}
	}

	switch req.Op {
	case OpLoadDefaults:
		cfg, err := openclaw.LoadConfig()
		if err != nil {
			return nil, err
		}
		resp.Config = cfg

	case OpRoundTrip:
		cfg := buildConfig(req)
		if err := openclaw.SaveConfig(cfg); err != nil {
			return nil, err
		}
		loaded, err := openclaw.LoadConfig()
		if err != nil {
			return nil, err
		}
		resp.Config = loaded
		resp.ConfigOnDisk = readConfigFromDisk(dir)

	case OpAPIGetConfig:
		code, body := apiCall(mux(), http.MethodGet, "/api/openclaw/config", "")
		resp.APIStatusCode = code
		resp.APIBody = body

	case OpAPIPutConfig:
		code, body := apiCall(mux(), http.MethodPut, "/api/openclaw/config", req.PutBody)
		resp.APIStatusCode = code
		resp.APIBody = body
		loaded, err := openclaw.LoadConfig()
		if err != nil {
			return nil, err
		}
		resp.Config = loaded
		resp.ConfigOnDisk = readConfigFromDisk(dir)

	case OpValidate:
		cfg, err := openclaw.LoadConfig()
		if err != nil {
			return nil, err
		}
		resp.Config = cfg
		resp.ValidationErr = openclaw.ValidateStartConfig(cfg)

	case OpStart:
		resp.StartErr = m.Start()
		resp.State, _ = openclaw.LoadState()
		resp.Status = m.Status()
		resp.GeneratedPath = generatedPath(dir)
		if data, err := os.ReadFile(resp.GeneratedPath); err == nil {
			resp.RenderedRaw = data
			_ = json.Unmarshal(data, &resp.RenderedJSON)
		}

	case OpAPIStart:
		path := "/api/openclaw/start"
		code, body := apiCall(mux(), http.MethodPost, path, "")
		resp.APIStatusCode = code
		resp.APIBody = body
		resp.State, _ = openclaw.LoadState()
		resp.Status = m.Status()
		if req.SecondStart {
			code2, body2 := apiCall(mux(), http.MethodPost, path, "")
			resp.SecondStartErr = parseAPIError(code2, body2)
			resp.APIStatusCode = code2
			resp.APIBody = body2
		}

	case OpStop:
		if err := m.Stop(); err != nil {
			return nil, err
		}
		resp.State, _ = openclaw.LoadState()
		resp.Status = m.Status()
		if req.SecondStart {
			if err := m.Stop(); err != nil {
				return nil, err
			}
			resp.State, _ = openclaw.LoadState()
		}

	case OpStatus:
		resp.Status = m.Status()

	case OpDryRun:
		dr, err := m.DryRun()
		if err != nil {
			return nil, err
		}
		resp.DryRun = dr
		resp.State, _ = openclaw.LoadState()

	case OpAPIDryRun:
		code, body := apiCall(mux(), http.MethodPost, "/api/openclaw/start?dry_run=true", "")
		resp.APIStatusCode = code
		resp.APIBody = body

	case OpRender:
		cfg, err := openclaw.LoadConfig()
		if err != nil {
			return nil, err
		}
		data, err := openclaw.RenderGatewayConfig(cfg)
		if err != nil {
			return nil, err
		}
		resp.RenderedRaw = data
		_ = json.Unmarshal(data, &resp.RenderedJSON)

	case OpDoctor:
		resp.Doctor = m.Doctor()

	default:
		t.Fatalf("unknown Op %q", req.Op)
	}

	return resp, nil
}

func mux() *http.ServeMux {
	m := http.NewServeMux()
	openclaw.RegisterAPI(m)
	return m
}

func apiCall(m *http.ServeMux, method, path, body string) (int, string) {
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	rec := httptest.NewRecorder()
	m.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

func buildConfig(req *Request) *openclaw.Config {
	cfg := &openclaw.Config{
		GatewayPort: req.GatewayPort,
		Workspace:   req.Workspace,
		Model:       req.Model,
	}
	if cfg.GatewayPort == 0 {
		cfg.GatewayPort = 18789
	}
	if req.SlackEnabled || req.BotToken != "" || req.AppToken != "" || req.SlackMode != "" {
		cfg.Slack = &openclaw.SlackConfig{
			Enabled:        req.SlackEnabled,
			Mode:           req.SlackMode,
			BotToken:       req.BotToken,
			AppToken:       req.AppToken,
			DMPolicy:       req.DMPolicy,
			AllowFrom:      req.AllowFrom,
			RequireMention: req.RequireMention,
		}
	}
	return cfg
}

func readConfigFromDisk(dir string) *openclaw.Config {
	data, err := os.ReadFile(dir + "/openclaw.json")
	if err != nil {
		return nil
	}
	cfg := &openclaw.Config{}
	_ = json.Unmarshal(data, cfg)
	return cfg
}

func generatedPath(dir string) string {
	return dir + "/openclaw/openclaw.json"
}

func parseAPIError(code int, body string) error {
	if code < 400 {
		return nil
	}
	var wrapper struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.Unmarshal([]byte(body), &wrapper)
	if wrapper.Error.Message != "" {
		return &openclaw.APIError{Code: openclaw.ErrorCode(wrapper.Error.Code), Message: wrapper.Error.Message}
	}
	return &openclaw.APIError{Code: openclaw.ErrInternal, Message: body}
}

func doctorCheck(report *openclaw.DoctorReport, id string) *openclaw.DoctorCheck {
	for i := range report.Checks {
		if report.Checks[i].ID == id {
			return &report.Checks[i]
		}
	}
	return nil
}

func apiErrorCode(body string) string {
	var wrapper struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal([]byte(body), &wrapper)
	return wrapper.Error.Code
}

func boolPtr(v bool) *bool { return &v }
```