# ws-proxy sing-box Doctests

Client-side doctest harness for `remote-agent ws-proxy sing-box` — TUN-ready
config generation from live VMess params and sing-box runtime orchestration
with injectable hooks (no real `brew`, `sudo`, or `sing-box`).

# DSN (Domain Specific Notion)

The ws-proxy sing-box harness models the macOS CLI path from ai-critic server
VMess params to a local TUN tunnel via sing-box.

**Participants**

- **remote-agent CLI** — `ws-proxy sing-box client-config` and `run-tun` subcommands.
- **API client** — `GET /api/ws-proxy/vmess-link`; requires ws-proxy client-ready.
- **Config builder** — pure `BuildSingBoxTunConfig` rendering tun inbound, vmess+ws
  outbound, DNS, route (LAN bypass → direct, final → proxy).
- **Hook layer** — injectable `LookPath`, `IsTTY`, `Confirm`, `BrewInstall`, `Geteuid`,
  `RunSingBox`, `StartDetached`, `FetchVMess`, `UserCacheDir`.
- **sing-box process** — foreground `sing-box run -c` (optionally via `sudo`) or
  detached background with PID + log under `~/.cache/remote-agent/singbox/`.
- **Homebrew** — optional `brew install sing-box` on TTY when binary missing (macOS).

**Behaviors**

- `client-config` fetches VMess params, emits valid sing-box JSON to stdout or `--output`.
- `run-tun` resolves config from `--config FILE` or fetch+build; ensures sing-box on PATH.
- Missing sing-box + non-TTY → error with `sing-box not installed` and brew hint.
- Missing sing-box + TTY → confirm prompt; `--yes` skips confirm; `--no-install` fails fast.
- euid≠0 → `sudo sing-box run`; euid=0 → direct; non-TTY + needs sudo → error.
- `--detach` → `StartDetached`, prints PID + config + log paths, parent exits 0.

## Version

0.0.2

## Decision Tree

```
[ws-proxy sing-box]
 |
 +-- client-config/                              (grouping: config emission only)
 |    +-- success/                               (grouping: API returns VMess)
 |    |    +-- emits-valid-tun-json/            (LEAF) stdout JSON, tun+vmess
 |    |    +-- writes-output-file/              (LEAF) --output FILE, quiet stdout
 |    +-- ws-proxy-not-ready/                   (LEAF) FetchVMess NOT_RUNNING error
 |
 +-- run-tun/                                    (grouping: sing-box execution)
 |    +-- foreground/                            (grouping: blocking run)
 |    |    +-- from-api/                         (grouping: fetch+build config)
 |    |    |    +-- present/                     (grouping: sing-box on PATH)
 |    |    |    |    +-- sing-box-present-no-sudo/     (LEAF) euid≠0 → sudo run
 |    |    |    |    +-- sing-box-present-as-root/     (LEAF) euid=0 → direct run
 |    |    |    |    +-- non-tty-needs-sudo-errors/    (LEAF) euid≠0, !TTY → error
 |    |    |    +-- missing/                     (grouping: sing-box absent)
 |    |    |         +-- missing-non-tty-errors/       (LEAF) !TTY → install error
 |    |    |         +-- missing-tty-decline/          (LEAF) TTY, user declines
 |    |    |         +-- missing-tty-accept-brew/      (LEAF) TTY, confirm yes → brew
 |    |    |         +-- missing-yes-skips-confirm/    (LEAF) TTY, --yes → brew, no prompt
 |    |    |         +-- no-install-flag/              (LEAF) --no-install, no brew
 |    |    +-- existing-config/                  (grouping: --config FILE)
 |    |         +-- uses-existing-config/         (LEAF) skips FetchVMess, uses file
 |    +-- detach/                                (grouping: background start)
 |         +-- starts-background-non-root/       (LEAF) StartDetached useSudo=true
 |         +-- starts-background-as-root/        (LEAF) StartDetached useSudo=false
 |
 +-- config-builder/                             (grouping: pure BuildSingBoxTunConfig)
      +-- vmess-ws-tls-fields/                   (LEAF) outbound server/port/path/tls
      +-- tun-inbound-defaults/                  (LEAF) auto_route, strict_route true
      +-- lan-bypass-routes/                     (LEAF) private CIDRs → direct
      +-- tls-disabled-edge/                     (LEAF) tls:"none" → tls.enabled false
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `client-config/success/emits-valid-tun-json` | Valid TUN JSON on stdout from mock VMess |
| 2 | `client-config/success/writes-output-file` | `--output` writes file; stdout minimal |
| 3 | `client-config/ws-proxy-not-ready` | API not-running error surfaces to CLI |
| 4 | `run-tun/foreground/from-api/present/sing-box-present-no-sudo` | Non-root foreground invokes sudo sing-box |
| 5 | `run-tun/foreground/from-api/present/sing-box-present-as-root` | Root foreground runs sing-box directly |
| 6 | `run-tun/foreground/from-api/present/non-tty-needs-sudo-errors` | Non-TTY non-root cannot sudo |
| 7 | `run-tun/foreground/from-api/missing/missing-non-tty-errors` | Missing binary, non-TTY install error |
| 8 | `run-tun/foreground/from-api/missing/missing-tty-decline` | TTY decline aborts before brew |
| 9 | `run-tun/foreground/from-api/missing/missing-tty-accept-brew` | TTY accept runs brew then sing-box |
| 10 | `run-tun/foreground/from-api/missing/missing-yes-skips-confirm` | `--yes` brews without confirm prompt |
| 11 | `run-tun/foreground/from-api/missing/no-install-flag` | `--no-install` fails fast, no brew |
| 12 | `run-tun/foreground/existing-config/uses-existing-config` | `--config` skips FetchVMess |
| 13 | `run-tun/detach/starts-background-non-root` | Detach prints PID, paths; useSudo=true |
| 14 | `run-tun/detach/starts-background-as-root` | Detach as root; useSudo=false |
| 15 | `config-builder/vmess-ws-tls-fields` | VMess WS+TLS outbound golden fields |
| 16 | `config-builder/tun-inbound-defaults` | TUN inbound auto_route/strict_route |
| 17 | `config-builder/lan-bypass-routes` | LAN CIDR rules route to direct |
| 18 | `config-builder/tls-disabled-edge` | tls:"none" disables TLS on outbound |

## How to Run

```sh
doctest vet ./cmd/remote-agent/wsproxy_singbox/tests
doctest test ./cmd/remote-agent/wsproxy_singbox/tests/...
go test ./cmd/remote-agent/... -run SingBox -count=1
go run ./script/build
```

```go
import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhd2015/ai-critic/client"
	singbox "github.com/xhd2015/ai-critic/cmd/remote-agent/wsproxy_singbox"
)

const (
	OpClientConfig = "client-config"
	OpRunTun       = "run-tun"
	OpBuildConfig  = "build-config"
)

type Request struct {
	Op string

	OutputFile string
	ConfigFile string
	Yes        bool
	NoInstall  bool
	Detach     bool

	MockVMess     *singbox.VMessParams
	FetchVMessErr error

	SingBoxOnPath bool
	IsTTY         bool
	ConfirmYes    *bool
	EUID          *int
	BrewInstallErr error
	DetachPID     int

	VMessFixture string
	BuildVMess   *singbox.VMessParams
}

type Response struct {
	Stdout string
	Stderr string
	RunErr error

	OutputPath     string
	OutputData     []byte
	ConfigJSON     map[string]any
	ConfigRaw      []byte
	CacheConfigPath string
	CacheLogPath    string

	FetchVMessCalled    bool
	BrewInstallCalled   bool
	ConfirmCalled       bool
	ConfirmPrompt       string
	RunSingBoxCalled    bool
	RunSingBoxSudo      bool
	RunSingBoxConfig    string
	StartDetachedCalled bool
	StartDetachedSudo   bool
	StartDetachedPID    int
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	cacheDir := filepath.Join(t.TempDir(), "singbox-cache")
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return nil, err
	}

	audit := newHookAudit(resp)
	restore := singbox.InstallTestHooks(singbox.TestHooks{
		LookPath: func(name string) (string, error) {
			if name == "sing-box" && req.SingBoxOnPath {
				return "/mock/sing-box", nil
			}
			return "", errors.New("executable not found")
		},
		IsTTY: func() bool { return req.IsTTY },
		Confirm: func(prompt string) bool {
			resp.ConfirmCalled = true
			resp.ConfirmPrompt = prompt
			if req.ConfirmYes != nil {
				return *req.ConfirmYes
			}
			return true
		},
		BrewInstall: func() error {
			resp.BrewInstallCalled = true
			return req.BrewInstallErr
		},
		Geteuid: func() int {
			if req.EUID != nil {
				return *req.EUID
			}
			return 1000
		},
		RunSingBox: func(ctx context.Context, sudo bool, configPath string) error {
			resp.RunSingBoxCalled = true
			resp.RunSingBoxSudo = sudo
			resp.RunSingBoxConfig = configPath
			return nil
		},
		StartDetached: func(configPath, logPath string, useSudo bool) (int, error) {
			resp.StartDetachedCalled = true
			resp.StartDetachedSudo = useSudo
			pid := req.DetachPID
			if pid == 0 {
				pid = 4242
			}
			resp.StartDetachedPID = pid
			resp.CacheConfigPath = configPath
			resp.CacheLogPath = logPath
			return pid, nil
		},
		FetchVMess: func(c *client.Client) (*singbox.VMessParams, error) {
			resp.FetchVMessCalled = true
			if req.FetchVMessErr != nil {
				return nil, req.FetchVMessErr
			}
			if req.MockVMess != nil {
				return req.MockVMess, nil
			}
			return defaultMockVMess(), nil
		},
		UserCacheDir: func() (string, error) { return cacheDir, nil },
	})
	defer restore()

	getClient := func() (*client.Client, error) {
		return client.New("http://mock-server", "test-token"), nil
	}

	var outBuf, errBuf bytes.Buffer
	oldOut, oldErr := os.Stdout, os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout, os.Stderr = wOut, wErr
	outDone := make(chan struct{})
	errDone := make(chan struct{})
	go func() {
		_, _ = io.Copy(&outBuf, rOut)
		close(outDone)
	}()
	go func() {
		_, _ = io.Copy(&errBuf, rErr)
		close(errDone)
	}()

	var runErr error
	switch req.Op {
	case OpClientConfig:
		runErr = singbox.RunClientConfig(getClient, singbox.ClientConfigOptions{
			OutputFile: req.OutputFile,
		})
	case OpRunTun:
		runErr = singbox.RunTun(getClient, singbox.RunTunOptions{
			ConfigFile: req.ConfigFile,
			Yes:        req.Yes,
			NoInstall:  req.NoInstall,
			Detach:     req.Detach,
		})
	case OpBuildConfig:
		vmess := req.BuildVMess
		if vmess == nil {
			vmess = loadVMessFixture(t, req.VMessFixture)
		}
		data, err := singbox.BuildSingBoxTunConfig(vmess, nil)
		if err != nil {
			runErr = err
		} else {
			resp.ConfigRaw = data
			_ = json.Unmarshal(data, &resp.ConfigJSON)
		}
	default:
		t.Fatalf("unknown Op %q", req.Op)
	}

	wOut.Close()
	wErr.Close()
	<-outDone
	<-errDone
	os.Stdout, os.Stderr = oldOut, oldErr

	resp.Stdout = outBuf.String()
	resp.Stderr = errBuf.String()
	resp.RunErr = runErr

	if req.OutputFile != "" {
		resp.OutputPath = req.OutputFile
		resp.OutputData, _ = os.ReadFile(req.OutputFile)
		if resp.ConfigRaw == nil && len(resp.OutputData) > 0 {
			resp.ConfigRaw = resp.OutputData
			_ = json.Unmarshal(resp.OutputData, &resp.ConfigJSON)
		}
	}

	_ = audit
	return resp, nil
}

func defaultMockVMess() *singbox.VMessParams {
	return &singbox.VMessParams{
		VMessLink: "vmess://eyJ2IjoiMiIsInBzIjoidGVzdCJ9",
		Host:      "ws-test.example.com",
		Port:      "443",
		UUID:      "11111111-2222-4333-8444-555555555555",
		AlterID:   "0",
		Network:   "ws",
		Type:      "none",
		Path:      "/ws",
		TLS:       "tls",
	}
}

func loadVMessFixture(t *testing.T, path string) *singbox.VMessParams {
	t.Helper()
	if path == "" {
		return defaultMockVMess()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read vmess fixture: %v", err)
	}
	var vmess singbox.VMessParams
	if err := json.Unmarshal(data, &vmess); err != nil {
		t.Fatalf("parse vmess fixture: %v", err)
	}
	return &vmess
}

type hookAudit struct{}

func newHookAudit(resp *Response) *hookAudit { return &hookAudit{} }

func configHasTunInbound(cfg map[string]any) bool {
	inbounds, ok := cfg["inbounds"].([]any)
	if !ok {
		return false
	}
	for _, in := range inbounds {
		m, ok := in.(map[string]any)
		if ok && m["type"] == "tun" {
			return true
		}
	}
	return false
}

func configHasVMessOutbound(cfg map[string]any) bool {
	outbounds, ok := cfg["outbounds"].([]any)
	if !ok {
		return false
	}
	for _, out := range outbounds {
		m, ok := out.(map[string]any)
		if ok && m["type"] == "vmess" {
			return true
		}
	}
	return false
}

func findOutbound(cfg map[string]any, typ string) map[string]any {
	outbounds, ok := cfg["outbounds"].([]any)
	if !ok {
		return nil
	}
	for _, out := range outbounds {
		m, ok := out.(map[string]any)
		if ok && m["type"] == typ {
			return m
		}
	}
	return nil
}

func findTunInbound(cfg map[string]any) map[string]any {
	inbounds, ok := cfg["inbounds"].([]any)
	if !ok {
		return nil
	}
	for _, in := range inbounds {
		m, ok := in.(map[string]any)
		if ok && m["type"] == "tun" {
			return m
		}
	}
	return nil
}

func routeRules(cfg map[string]any) []map[string]any {
	route, ok := cfg["route"].(map[string]any)
	if !ok {
		return nil
	}
	rules, ok := route["rules"].([]any)
	if !ok {
		return nil
	}
	var out []map[string]any
	for _, r := range rules {
		if m, ok := r.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func errContains(err error, subs ...string) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, s := range subs {
		if !strings.Contains(msg, strings.ToLower(s)) {
			return false
		}
	}
	return true
}
```