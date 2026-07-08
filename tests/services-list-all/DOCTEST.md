# Services List All API Doctests

Server API tests for `GET /api/services` with optional `?all=1` to return every
managed service definition regardless of project scope.

# DSN (Domain Specific Notion)

**Participants**

- **ai-critic-server subprocess** — loads `{AI_CRITIC_HOME}/services.json`, exposes
  `GET /api/services` on the main server port (`23712` by default).
- **Service definitions** — rows in `services.json` with optional `projectDir` scoping.
- **Services manager (`server/services`)** — `List(projectDir)` filters by project;
  `?all=1` bypasses scope and returns all definitions.
- **HTTP client** — authenticated `GET /api/services` with and without `all=1`.
- **Test harness** — seeds multi-project `services.json`, starts isolated server,
  asserts listed service IDs.

**Behaviors**

- Default `GET /api/services` returns only services matching the server's project scope.
- `GET /api/services?all=1` returns services across all `projectDir` values.
- Response is JSON array of `ServiceStatus` objects with `id` fields.

## Version

0.0.2

## Decision Tree

```
[services list API]
 |
 +-- list-scoped-default/             (LEAF)   without all=1, project-scoped only
 +-- list-all/                        (LEAF)   ?all=1 returns cross-scope services
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `list-scoped-default` | Default list excludes other-project services |
| 2 | `list-all` | `?all=1` includes services from all project dirs |

## Parameter Coverage

| Leaf | Query | Seeded projects | Expect |
|------|-------|-----------------|--------|
| list-scoped-default | none | local + other | only local ID |
| list-all | `all=1` | local + other | both IDs |

## How to Run

```sh
doctest vet ./tests/services-list-all
doctest test ./tests/services-list-all/...
```

```go
import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/script/lib"
)

type ServiceSeed struct {
	ID         string
	Name       string
	Command    string
	ProjectDir string
}

type Request struct {
	Op string // list-scoped | list-all

	LocalProjectDir  string
	OtherProjectDir  string
	LocalServiceID   string
	OtherServiceID   string

	Token      string
	ServerPort int
}

type serviceStatus struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	ProjectDir string `json:"projectDir,omitempty"`
}

type Response struct {
	ServerPort int
	ConfigHome string
	ListedIDs  []string
	HTTPStatus int
	Body       string
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	if req.Token == "" {
		req.Token = lib.TestPassword
	}
	if req.LocalServiceID == "" {
		req.LocalServiceID = "local-web"
	}
	if req.OtherServiceID == "" {
		req.OtherServiceID = "other-api"
	}

	moduleRoot, err := findModuleRoot()
	if err != nil {
		return nil, err
	}

	safeName := strings.ReplaceAll(t.Name(), "/", "_")
	serverBin := filepath.Join(os.TempDir(), "ai-critic-services-list-all-"+safeName)
	build := exec.Command("go", "build", "-o", serverBin, ".")
	build.Dir = moduleRoot
	if out, err := build.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("build server: %w\n%s", err, string(out))
	}
	t.Cleanup(func() { os.Remove(serverBin) })

	configHome, err := lib.CreateTestConfigHome()
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(configHome) })
	resp.ConfigHome = configHome

	localDir := req.LocalProjectDir
	if localDir == "" {
		localDir = configHome
	}
	otherDir := req.OtherProjectDir
	if otherDir == "" {
		otherDir = filepath.Join(configHome, "other-project")
	}
	if err := os.MkdirAll(otherDir, 0755); err != nil {
		return nil, err
	}

	if err := writeServicesJSON(configHome, []ServiceSeed{
		{ID: req.LocalServiceID, Name: "web", Command: "sleep 300", ProjectDir: localDir},
		{ID: req.OtherServiceID, Name: "api", Command: "sleep 300", ProjectDir: otherDir},
	}); err != nil {
		return nil, err
	}

	portBase := portBaseFromTestName(t.Name())
	serverPort, err := pickFreePort(portBase)
	if err != nil {
		return nil, err
	}
	resp.ServerPort = serverPort

	credFile, err := lib.WriteTestCredentials(configHome)
	if err != nil {
		return nil, err
	}

	serverCmd := exec.Command(serverBin, "--port", strconv.Itoa(serverPort), "--credentials-file", credFile)
	serverCmd.Dir = localDir
	serverCmd.Env = lib.AppendTestServerEnv(os.Environ(), configHome)
	if err := serverCmd.Start(); err != nil {
		return nil, fmt.Errorf("start server: %w", err)
	}
	t.Cleanup(func() {
		if serverCmd.Process != nil {
			serverCmd.Process.Signal(syscall.SIGTERM)
			time.Sleep(150 * time.Millisecond)
			serverCmd.Process.Kill()
		}
	})

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", serverPort)
	if err := waitHTTPReady(baseURL+"/ping", 30*time.Second); err != nil {
		return nil, err
	}

	query := ""
	switch req.Op {
	case "list-scoped":
		query = ""
	case "list-all":
		query = "all=1"
	default:
		return nil, fmt.Errorf("unknown op %q", req.Op)
	}

	ids, status, body, err := getServiceIDs(baseURL, req.Token, query)
	if err != nil {
		return nil, err
	}
	resp.ListedIDs = ids
	resp.HTTPStatus = status
	resp.Body = body
	return resp, nil
}

func writeServicesJSON(configHome string, services []ServiceSeed) error {
	now := "2026-07-07T00:00:00Z"
	rows := make([]map[string]any, 0, len(services))
	for _, svc := range services {
		row := map[string]any{
			"id":        svc.ID,
			"name":      svc.Name,
			"command":   svc.Command,
			"createdAt": now,
			"updatedAt": now,
		}
		if svc.ProjectDir != "" {
			row["projectDir"] = svc.ProjectDir
		}
		rows = append(rows, row)
	}
	data, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(configHome, "services.json"), data, 0644)
}

func getServiceIDs(baseURL, token, query string) ([]string, int, string, error) {
	url := baseURL + "/api/services"
	if query != "" {
		url += "?" + query
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, string(body), fmt.Errorf("GET %s status %d: %s", url, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var services []serviceStatus
	if err := json.Unmarshal(body, &services); err != nil {
		return nil, resp.StatusCode, string(body), err
	}
	ids := make([]string, 0, len(services))
	for _, svc := range services {
		ids = append(ids, svc.ID)
	}
	return ids, resp.StatusCode, string(body), nil
}

func waitHTTPReady(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s", url)
}

func pickFreePort(base int) (int, error) {
	for port := base; port < base+200; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			ln.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("no free port near %d", base)
}

func portBaseFromTestName(name string) int {
	h := 0
	for _, c := range name {
		h = h*31 + int(c)
	}
	if h < 0 {
		h = -h
	}
	return 24000 + (h % 500)
}

func findModuleRoot() (string, error) {
	if root := os.Getenv("DOCTEST_ROOT"); root != "" {
		for dir := root; ; dir = filepath.Dir(dir) {
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