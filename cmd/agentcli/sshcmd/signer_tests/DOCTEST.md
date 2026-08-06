# remote-agent ssh — L2 doctests (Signer persist + load wire)

Classic TDD (gap fix): persist client ed25519 under `configDir`, reload stable
identity, and wire `CryptoSSHRunner.Signer` from disk so client run no longer
fails with `ssh signer not configured` when a key exists and the session is Alive.

Sibling of sealed P1–P4 trees under `cmd/agentcli/sshcmd/` — **do not edit**
`tests/`, `session_tests/`, `ssh_tests/`, or `tunnel_tests/` ASSERT files.

Package under test (new symbols **RED** until implementer):

```text
github.com/xhd2015/ai-critic/cmd/agentcli/sshcmd
  EnsureClientKeyPair(configDir) (*ClientKeyPair, error)
  # ClientKeyPair, GenerateClientKeyPair, CryptoSSHRunner, AdhocServer, ServeService (existing)
```

Out of scope: host key pin polish; Unison e2e; multi-profile; server API changes;
agentcli HOME/env wiring (optional thin wrapper may call the same helper later).

Data layout under test:

```text
{configDir}/id_ed25519      # private, mode 0600, ssh.ParsePrivateKey-able
{configDir}/id_ed25519.pub  # optional OpenSSH public line
```

Network model (load-from-disk leaf only):

```text
EnsureClientKeyPair(configDir) -> Signer + Public
AdhocServer(authorized=Public) :R
ServeService{Dial: TCP(Adhoc)} -> LocalRelay :L  (Alive session)
CryptoSSHRunner{Signer: from EnsureClientKeyPair} -> :L -> Adhoc
  -> echo signer-ok in stdout
```

# DSN (Domain Specific Notion)

**remote-agent ssh Signer wire** closes the remaining client-auth gap: one helper
owns key material on disk; serve and runner both consume that identity.

**Participants**

- **EnsureClientKeyPair** — load `id_ed25519` from `configDir` or generate, write
  private 0600 (+ optional `.pub`), return `*ClientKeyPair`.
- **ClientKeyPair** — existing `{Signer, Public}`; private parseable by
  `ssh.ParsePrivateKey`.
- **CryptoSSHRunner** — existing; requires non-nil `Signer` or errors with
  message containing `signer`.
- **AdhocServer + ServeService + LocalRelay** — existing P2/P3 compose for the
  end-to-end echo leaf (no remote agent tunnel required).

**Behaviors**

- Missing key → create under `configDir`; Signer and Public non-nil; private
  file mode owner-only write (0600).
- Second Ensure same dir → same public key material (stable identity).
- Corrupt private file → error (do not silently regenerate).
- Runner with nil Signer → error containing `signer` (regression: wire-up required).
- Alive session + key on disk: load Signer via Ensure, run remote echo through
  serve+adhoc → stdout contains needle.

## Version

0.0.2

## Decision Tree

```
cmd/agentcli/sshcmd/signer_tests/     [Request{Scenario, ConfigDir, …}]
│                                      Run dispatches by Scenario (L2 library)
├── ensure-key-pair/                   # EnsureClientKeyPair disk lifecycle
│   ├── create-missing/                # empty dir → create id_ed25519 0600
│   ├── reload-stable/                 # second call → same public material
│   └── corrupt-private/               # garbage private → error
├── crypto-runner/                     # CryptoSSHRunner Signer gate
│   └── nil-signer/                    # nil Signer → error contains "signer"
└── through-relay/                     # compose: Ensure → Adhoc → Serve → Run
    └── load-from-disk/                # Signer from Ensure only; echo signer-ok
```

**Significance order:** surface (ensure-key-pair | crypto-runner | through-relay)
→ disk outcome / gate / compose within surface.

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `ensure-key-pair/create-missing` | Missing key → create `id_ed25519` mode 0600; Signer+Public set |
| 2 | `ensure-key-pair/reload-stable` | Two Ensure calls same dir → identical public key material |
| 3 | `ensure-key-pair/corrupt-private` | Corrupt `id_ed25519` → Ensure error (no silent regen) |
| 4 | `crypto-runner/nil-signer` | `CryptoSSHRunner` with nil Signer → error contains `signer` |
| 5 | `through-relay/load-from-disk` | Ensure loads Signer from disk; serve+adhoc `echo signer-ok` |

## Exported APIs (implementer contract)

Package `github.com/xhd2015/ai-critic/cmd/agentcli/sshcmd`:

| Symbol | Role |
|--------|------|
| `EnsureClientKeyPair` | `(configDir string) (*ClientKeyPair, error)` — **new**. If `{configDir}/id_ed25519` exists and parses via `ssh.ParsePrivateKey`, return that pair. Else generate ed25519, write private to `id_ed25519` with mode **0600**, optionally write `id_ed25519.pub` (OpenSSH authorized_keys line), return pair. Creates `configDir` if missing (`0755` OK). Corrupt / unreadable private → **error** (prefer fail-closed; do not overwrite). |
| `ClientKeyPair` | existing `struct { Signer ssh.Signer; Public ssh.PublicKey }` |
| `GenerateClientKeyPair` | existing in-memory generator (Ensure may use internally) |
| `CryptoSSHRunner` | existing; `Signer` must be set by production wire from Ensure |
| `(*CryptoSSHRunner).Run` | existing; nil Signer → `"ssh signer not configured"` (contains `signer`) |

Private key on disk must be parseable by `ssh.ParsePrivateKey` (OpenSSH or PEM).
Suggested write path: `ssh.MarshalPrivateKey` (or equivalent) → `os.WriteFile(path, pem, 0o600)`.

Production wiring (implementer; not asserted by every leaf, documented here):

```go
// client run / runSSH:
kp, err := sshcmd.EnsureClientKeyPair(configDir)
runner := &sshcmd.CryptoSSHRunner{Signer: kp.Signer, /* stdio, InsecureIgnoreHostKey */}

// --serve Start (same helper, same identity as CreateSession public key):
kp, err := sshcmd.EnsureClientKeyPair(configDir)
// pub := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(kp.Public)))
// BuildSSHTunnelDial(client, pub); ServeService{ConfigDir: configDir, …}
```

Optional: replace unexported `agentcli.loadOrGenerateServeKey` with a call to
`sshcmd.EnsureClientKeyPair` so serve and client share one identity.

## How to Run

From module root (`external/ai-critic-master-2026-07-31`):

```sh
doctest vet ./cmd/agentcli/sshcmd/signer_tests
doctest test ./cmd/agentcli/sshcmd/signer_tests
# Sealed siblings must stay green:
doctest test ./cmd/agentcli/sshcmd/tests
doctest test ./cmd/agentcli/sshcmd/session_tests
doctest test ./cmd/agentcli/sshcmd/ssh_tests
doctest test ./cmd/agentcli/sshcmd/tunnel_tests
```

```go
import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/cmd/agentcli/sshcmd"
	"github.com/xhd2015/doctest/session"
	"golang.org/x/crypto/ssh"
)

// Scenario selects which Signer-wire surface Run exercises (MECE dispatch key).
type Scenario string

const (
	ScenarioEnsureCreateMissing Scenario = "ensure-create-missing"
	ScenarioEnsureReloadStable  Scenario = "ensure-reload-stable"
	ScenarioEnsureCorrupt       Scenario = "ensure-corrupt-private"
	ScenarioCryptoNilSigner     Scenario = "crypto-nil-signer"
	ScenarioRelayLoadFromDisk   Scenario = "through-relay-load-from-disk"
)

// Request configures absolute paths and scenario inputs (parallel-safe).
// All paths under d.DOCTEST_CASE — no Setenv/Chdir.
type Request struct {
	Scenario  Scenario
	ProfileID string
	// Root is FileSessionStore root ({Root}/ssh-sessions/…).
	Root string
	// ConfigDir holds id_ed25519 (Ensure target and session ConfigDir).
	ConfigDir string

	// RemoteArgv / EchoNeedle for through-relay load-from-disk.
	RemoteArgv []string
	EchoNeedle string

	// CorruptPrivateBytes, when non-empty, is written to id_ed25519 before
	// Ensure (corrupt-private leaf). Default "not-a-valid-ssh-private-key\n".
	CorruptPrivateBytes []byte
}

// Response captures Ensure, runner gate, and compose outcomes.
type Response struct {
	// EnsureClientKeyPair
	EnsureErr      string
	EnsureErr2     string // second call (reload-stable)
	PrivateKeyPath string
	PrivateExists  bool
	PrivateMode    os.FileMode // permission bits only (ModePerm)
	SignerNonNil   bool
	PublicNonNil   bool
	// PublicMaterial is OpenSSH authorized_keys line (trimmed) from first Ensure.
	PublicMaterial string
	// PublicMaterial2 is from second Ensure (reload-stable).
	PublicMaterial2 string
	// ReloadedSame is true when PublicMaterial == PublicMaterial2 and both non-empty.
	ReloadedSame bool

	// CryptoSSHRunner
	RunnerErr string
	Stdout    string
	Stderr    string

	// Through-relay compose
	ServeErr          string
	SessionAfterStart *sshcmd.Session
	RelayLocalPort    int
	AdhocPort         int
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	t.Helper()
	applyDefaults(d, req)

	resp := &Response{}
	switch req.Scenario {
	case ScenarioEnsureCreateMissing:
		runEnsureCreateMissing(t, d, req, resp)
	case ScenarioEnsureReloadStable:
		runEnsureReloadStable(t, d, req, resp)
	case ScenarioEnsureCorrupt:
		runEnsureCorrupt(t, d, req, resp)
	case ScenarioCryptoNilSigner:
		runCryptoNilSigner(t, d, req, resp)
	case ScenarioRelayLoadFromDisk:
		runThroughRelayLoadFromDisk(t, d, req, resp)
	default:
		return nil, fmt.Errorf("unknown scenario %q", req.Scenario)
	}
	return resp, nil
}

func applyDefaults(d *session.Doctest, req *Request) {
	if req.ProfileID == "" {
		req.ProfileID = "default"
	}
	if req.Root == "" {
		req.Root = filepath.Join(d.DOCTEST_CASE, "store-root")
	}
	if req.ConfigDir == "" {
		req.ConfigDir = filepath.Join(d.DOCTEST_CASE, "ssh-config")
	}
	if len(req.RemoteArgv) == 0 {
		req.RemoteArgv = []string{"echo", "signer-ok"}
	}
	if req.EchoNeedle == "" {
		req.EchoNeedle = "signer-ok"
	}
	if len(req.CorruptPrivateBytes) == 0 {
		req.CorruptPrivateBytes = []byte("not-a-valid-ssh-private-key\n")
	}
}

func privateKeyPath(configDir string) string {
	return filepath.Join(configDir, "id_ed25519")
}

func publicMaterial(pub ssh.PublicKey) string {
	if pub == nil {
		return ""
	}
	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pub)))
}

func recordKeyPair(resp *Response, configDir string, kp *sshcmd.ClientKeyPair, which int) {
	resp.PrivateKeyPath = privateKeyPath(configDir)
	if st, err := os.Stat(resp.PrivateKeyPath); err == nil {
		resp.PrivateExists = true
		resp.PrivateMode = st.Mode().Perm()
	}
	if kp == nil {
		return
	}
	if which == 1 {
		resp.SignerNonNil = kp.Signer != nil
		resp.PublicNonNil = kp.Public != nil
		if kp.Public != nil {
			resp.PublicMaterial = publicMaterial(kp.Public)
		}
	} else if which == 2 {
		if kp.Public != nil {
			resp.PublicMaterial2 = publicMaterial(kp.Public)
		}
	}
}

// --- ensure leaves ---

func runEnsureCreateMissing(t *testing.T, d *session.Doctest, req *Request, resp *Response) {
	t.Helper()
	_ = d
	// ConfigDir starts empty (no id_ed25519).
	if _, err := os.Stat(privateKeyPath(req.ConfigDir)); err == nil {
		t.Fatalf("precondition: %s must not exist before create-missing", privateKeyPath(req.ConfigDir))
	}

	kp, err := sshcmd.EnsureClientKeyPair(req.ConfigDir)
	if err != nil {
		resp.EnsureErr = err.Error()
		return
	}
	recordKeyPair(resp, req.ConfigDir, kp, 1)
}

func runEnsureReloadStable(t *testing.T, d *session.Doctest, req *Request, resp *Response) {
	t.Helper()
	_ = d

	kp1, err := sshcmd.EnsureClientKeyPair(req.ConfigDir)
	if err != nil {
		resp.EnsureErr = err.Error()
		return
	}
	recordKeyPair(resp, req.ConfigDir, kp1, 1)

	kp2, err := sshcmd.EnsureClientKeyPair(req.ConfigDir)
	if err != nil {
		resp.EnsureErr2 = err.Error()
		return
	}
	recordKeyPair(resp, req.ConfigDir, kp2, 2)
	resp.ReloadedSame = resp.PublicMaterial != "" && resp.PublicMaterial == resp.PublicMaterial2
	// Second call still needs non-nil pair for ASSERT completeness.
	if kp2 != nil {
		resp.SignerNonNil = resp.SignerNonNil && kp2.Signer != nil
		resp.PublicNonNil = resp.PublicNonNil && kp2.Public != nil
	}
}

func runEnsureCorrupt(t *testing.T, d *session.Doctest, req *Request, resp *Response) {
	t.Helper()
	_ = d
	if err := os.MkdirAll(req.ConfigDir, 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	path := privateKeyPath(req.ConfigDir)
	if err := os.WriteFile(path, req.CorruptPrivateBytes, 0o600); err != nil {
		t.Fatalf("write corrupt private: %v", err)
	}
	resp.PrivateKeyPath = path
	resp.PrivateExists = true

	kp, err := sshcmd.EnsureClientKeyPair(req.ConfigDir)
	if err != nil {
		resp.EnsureErr = err.Error()
		// Fail-closed: must not return a usable pair on corrupt.
		if kp != nil {
			resp.SignerNonNil = kp.Signer != nil
			resp.PublicNonNil = kp.Public != nil
		}
		return
	}
	// Unexpected success: record pair so ASSERT can fail clearly.
	recordKeyPair(resp, req.ConfigDir, kp, 1)
}

// --- crypto-runner gate ---

func runCryptoNilSigner(t *testing.T, d *session.Doctest, req *Request, resp *Response) {
	t.Helper()
	_ = d
	// LocalPort > 0 so the nil-Signer check is the first gate (before dial).
	// No Adhoc needed: CryptoSSHRunner returns before network I/O.
	var stdout, stderr bytes.Buffer
	runner := &sshcmd.CryptoSSHRunner{
		Signer:                nil,
		Stdout:                &stdout,
		Stderr:                &stderr,
		InsecureIgnoreHostKey: true,
	}
	sess := &sshcmd.Session{
		LocalPort: 1,
		User:      "agent",
		ConfigDir: req.ConfigDir,
		ProfileID: req.ProfileID,
		Alive:     true,
	}
	err := runner.Run(sess, []string{"echo", "should-not-run"}, sshcmd.RunnerOpts{ProfileID: req.ProfileID})
	resp.Stdout = stdout.String()
	resp.Stderr = stderr.String()
	if err != nil {
		resp.RunnerErr = err.Error()
	}
}

// --- through-relay: Signer loaded only via EnsureClientKeyPair ---

func dialFuncTo(addr string) sshcmd.DialFunc {
	return func() (net.Conn, error) {
		return net.DialTimeout("tcp", addr, 5*time.Second)
	}
}

func runThroughRelayLoadFromDisk(t *testing.T, d *session.Doctest, req *Request, resp *Response) {
	t.Helper()
	_ = d
	if err := os.MkdirAll(req.ConfigDir, 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	if err := os.MkdirAll(req.Root, 0o755); err != nil {
		t.Fatalf("mkdir root: %v", err)
	}

	// Production wire: load/create key from configDir (not GenerateClientKeyPair).
	kp, err := sshcmd.EnsureClientKeyPair(req.ConfigDir)
	if err != nil {
		resp.EnsureErr = err.Error()
		return
	}
	recordKeyPair(resp, req.ConfigDir, kp, 1)
	if kp == nil || kp.Signer == nil || kp.Public == nil {
		return
	}

	// Second load: runner must use disk-backed identity (stable after Ensure).
	kpDisk, err := sshcmd.EnsureClientKeyPair(req.ConfigDir)
	if err != nil {
		resp.EnsureErr2 = err.Error()
		return
	}
	if kpDisk == nil || kpDisk.Signer == nil {
		resp.EnsureErr2 = "second EnsureClientKeyPair returned nil signer"
		return
	}
	resp.PublicMaterial2 = publicMaterial(kpDisk.Public)
	resp.ReloadedSame = resp.PublicMaterial != "" && resp.PublicMaterial == resp.PublicMaterial2

	adhoc := &sshcmd.AdhocServer{User: "agent", ForcePipeShell: true}
	adhoc.SetAuthorizedKeys([]ssh.PublicKey{kp.Public})
	if err := adhoc.Start(); err != nil {
		resp.ServeErr = "AdhocServer.Start: " + err.Error()
		return
	}
	t.Cleanup(func() { _ = adhoc.Close() })
	resp.AdhocPort = adhoc.LocalPort()

	store := &sshcmd.FileSessionStore{Root: req.Root}
	svc := &sshcmd.ServeService{
		Store:     store,
		ProfileID: req.ProfileID,
		Dial:      dialFuncTo(adhoc.Addr()),
		User:      "agent",
		ConfigDir: req.ConfigDir,
		ServePID:  os.Getpid(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var startErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		startErr = svc.Start(ctx)
	}()

	var afterStart *sshcmd.Session
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		loaded, err := store.Load(req.ProfileID)
		if err == nil && loaded != nil && loaded.Alive && loaded.LocalPort > 0 {
			afterStart = loaded
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	resp.SessionAfterStart = afterStart
	if afterStart == nil {
		cancel()
		wg.Wait()
		if startErr != nil {
			resp.ServeErr = startErr.Error()
		} else {
			resp.ServeErr = "timeout waiting for Alive session"
		}
		return
	}
	resp.RelayLocalPort = afterStart.LocalPort

	// Signer only from EnsureClientKeyPair (disk), never GenerateClientKeyPair.
	var stdout, stderr bytes.Buffer
	runner := &sshcmd.CryptoSSHRunner{
		Signer:                kpDisk.Signer,
		Stdout:                &stdout,
		Stderr:                &stderr,
		InsecureIgnoreHostKey: true,
	}
	runErr := runner.Run(afterStart, req.RemoteArgv, sshcmd.RunnerOpts{ProfileID: req.ProfileID})
	resp.Stdout = stdout.String()
	resp.Stderr = stderr.String()
	if runErr != nil {
		resp.RunnerErr = runErr.Error()
	}

	cancel()
	wg.Wait()
	if startErr != nil && startErr != context.Canceled {
		if !strings.Contains(startErr.Error(), "canceled") && !strings.Contains(startErr.Error(), "cancelled") {
			resp.ServeErr = startErr.Error()
		}
	}
}
```
