package services

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/xhd2015/ai-critic/server/proxy/portforward"
)

func TestEnsurePortForwardReplacesMismatchedForwardForSamePort(t *testing.T) {
	pfm := portforward.NewManager()
	provider := &testPortForwardProvider{name: portforward.ProviderCloudflareOwned}
	pfm.RegisterProvider(provider)

	if _, err := pfm.Add(9476, "knowledge-base-782as-sub-server-v2.xhd2015.xyz", portforward.ProviderCloudflareOwned); err != nil {
		t.Fatalf("Add stale forward error = %v", err)
	}

	m := &Manager{
		processes: map[string]*serviceProcess{
			"svc-v3": {},
		},
		portForwardManager: pfm,
	}

	err := m.ensurePortForward("svc-v3", ServiceDefinition{
		PortForward: &ServicePortForward{
			Port:       9476,
			Provider:   portforward.ProviderCloudflareOwned,
			BaseDomain: "xhd2015.xyz",
			Subdomain:  "knowledge-base-782as-sub-server-v3",
		},
	})
	if err != nil {
		t.Fatalf("ensurePortForward() error = %v", err)
	}

	forwards := pfm.List()
	if len(forwards) != 1 {
		t.Fatalf("forward count = %d, want 1: %#v", len(forwards), forwards)
	}
	got := forwards[0]
	if got.LocalPort != 9476 || got.Label != "knowledge-base-782as-sub-server-v3.xhd2015.xyz" {
		t.Fatalf("forward = port %d label %q, want 9476/v3", got.LocalPort, got.Label)
	}
	if provider.StopCount() != 1 {
		t.Fatalf("stale stop count = %d, want 1", provider.StopCount())
	}
	if !m.processes["svc-v3"].ownedForward {
		t.Fatalf("service did not take ownership of replacement forward")
	}
}

func TestEnsurePortForwardKeepsExistingServiceOwnership(t *testing.T) {
	pfm := portforward.NewManager()
	provider := &testPortForwardProvider{name: portforward.ProviderCloudflareOwned}
	pfm.RegisterProvider(provider)

	if _, err := pfm.Add(9476, "knowledge-base-782as-sub-server-v3.xhd2015.xyz", portforward.ProviderCloudflareOwned); err != nil {
		t.Fatalf("Add existing forward error = %v", err)
	}

	m := &Manager{
		processes: map[string]*serviceProcess{
			"svc-v3": {ownedForward: true},
		},
		portForwardManager: pfm,
	}

	err := m.ensurePortForward("svc-v3", ServiceDefinition{
		PortForward: &ServicePortForward{
			Port:       9476,
			Provider:   portforward.ProviderCloudflareOwned,
			BaseDomain: "xhd2015.xyz",
			Subdomain:  "knowledge-base-782as-sub-server-v3",
		},
	})
	if err != nil {
		t.Fatalf("ensurePortForward() error = %v", err)
	}
	if !m.processes["svc-v3"].ownedForward {
		t.Fatalf("service ownership was cleared for an existing matching forward")
	}
	if provider.StopCount() != 0 {
		t.Fatalf("stop count = %d, want 0", provider.StopCount())
	}
}

func TestServiceUpgradeTargetFallbackAndRemoteMemory(t *testing.T) {
	useTempServicesConfig(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	m := &Manager{
		definitions: []ServiceDefinition{
			{ID: "svc-1", Name: "web", Command: "sleep 10"},
		},
		processes: map[string]*serviceProcess{},
	}

	selected, err := m.selectServiceUpgradeTarget("svc-1", "server-linux-amd64", "")
	if err != nil {
		t.Fatalf("selectServiceUpgradeTarget() default error = %v", err)
	}
	if selected.Path != filepath.Join(home, "server-linux-amd64") {
		t.Fatalf("default target path = %q, want %q", selected.Path, filepath.Join(home, "server-linux-amd64"))
	}
	if selected.Remembered != "" {
		t.Fatalf("default remembered target = %q, want empty", selected.Remembered)
	}

	selected, err = m.selectServiceUpgradeTarget("svc-1", "server-linux-amd64", "~/bin/server")
	if err != nil {
		t.Fatalf("selectServiceUpgradeTarget() specified error = %v", err)
	}
	if selected.Path != filepath.Join(home, "bin", "server") {
		t.Fatalf("specified target path = %q, want %q", selected.Path, filepath.Join(home, "bin", "server"))
	}
	if selected.Remembered != "~/bin/server" {
		t.Fatalf("specified remembered target = %q, want ~/bin/server", selected.Remembered)
	}
	if m.definitions[0].UpgradeTarget != "~/bin/server" {
		t.Fatalf("service upgrade target = %q, want ~/bin/server", m.definitions[0].UpgradeTarget)
	}

	selected, err = m.selectServiceUpgradeTarget("svc-1", "server-linux-amd64", "")
	if err != nil {
		t.Fatalf("selectServiceUpgradeTarget() remembered error = %v", err)
	}
	if selected.Path != filepath.Join(home, "bin", "server") {
		t.Fatalf("remembered target path = %q, want %q", selected.Path, filepath.Join(home, "bin", "server"))
	}
	if selected.Remembered != "~/bin/server" {
		t.Fatalf("remembered target = %q, want ~/bin/server", selected.Remembered)
	}
}

func TestCreateOrUpdatePreservesServiceUpgradeTarget(t *testing.T) {
	useTempServicesConfig(t)

	m := &Manager{
		definitions: []ServiceDefinition{
			{ID: "svc-1", Name: "web", Command: "sleep 10", UpgradeTarget: "~/bin/server"},
		},
		processes: map[string]*serviceProcess{},
	}

	if _, err := m.CreateOrUpdate(ServiceDefinition{ID: "svc-1", Name: "web", Command: "sleep 20"}); err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if m.definitions[0].UpgradeTarget != "~/bin/server" {
		t.Fatalf("upgrade target after update = %q, want ~/bin/server", m.definitions[0].UpgradeTarget)
	}
}

func TestCreateOrUpdateNoRestartDoesNotMutateRunningProcessDefinition(t *testing.T) {
	useTempServicesConfig(t)

	oldDef := ServiceDefinition{ID: "svc-1", Name: "web", Command: "sleep 10", WorkingDir: "/old"}
	m := &Manager{
		definitions: []ServiceDefinition{oldDef},
		processes: map[string]*serviceProcess{
			"svc-1": {def: oldDef, desired: true},
		},
	}

	if _, err := m.CreateOrUpdateNoRestart(ServiceDefinition{
		ID:         "svc-1",
		Name:       "web",
		Command:    "sleep 20",
		WorkingDir: "/new",
	}); err != nil {
		t.Fatalf("CreateOrUpdateNoRestart() error = %v", err)
	}
	if m.definitions[0].Command != "sleep 20" || m.definitions[0].WorkingDir != "/new" {
		t.Fatalf("saved definition = command %q dir %q, want updated", m.definitions[0].Command, m.definitions[0].WorkingDir)
	}
	if proc := m.processes["svc-1"]; proc.def.Command != "sleep 10" || proc.def.WorkingDir != "/old" {
		t.Fatalf("running process definition changed to command %q dir %q", proc.def.Command, proc.def.WorkingDir)
	}
}

func TestEnsureServiceWorkingDirCreatesMissingDirectory(t *testing.T) {
	base := t.TempDir()
	workingDir := filepath.Join(base, "nested", "my-openclaw")

	if err := ensureServiceWorkingDir(workingDir); err != nil {
		t.Fatalf("ensureServiceWorkingDir() error = %v", err)
	}
	info, err := os.Stat(workingDir)
	if err != nil {
		t.Fatalf("Stat(workingDir) error = %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("working dir is not a directory")
	}
}

func TestEnsureServiceWorkingDirNoopsForEmptyPath(t *testing.T) {
	if err := ensureServiceWorkingDir(""); err != nil {
		t.Fatalf("ensureServiceWorkingDir(\"\") error = %v", err)
	}
}

func TestResolveServiceUpgradeTargetPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	tests := []struct {
		name   string
		target string
		want   string
	}{
		{name: "default relative", target: "server-linux-amd64", want: filepath.Join(home, "server-linux-amd64")},
		{name: "home shorthand", target: "~/bin/server", want: filepath.Join(home, "bin", "server")},
		{name: "home directory shorthand", target: "~/", want: filepath.Join(home, "server-linux-amd64")},
		{name: "absolute", target: "/opt/agent/server", want: "/opt/agent/server"},
		{name: "directory target", target: "/opt/agent/", want: "/opt/agent/server-linux-amd64"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveServiceUpgradeTargetPath(tt.target, "server-linux-amd64")
			if err != nil {
				t.Fatalf("resolveServiceUpgradeTargetPath() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("resolveServiceUpgradeTargetPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMoveServiceUpgradeFileSetsExecutableBit(t *testing.T) {
	dir := t.TempDir()
	tmpPath := filepath.Join(dir, "uploaded.tmp")
	targetPath := filepath.Join(dir, "bin", "server")
	if err := os.WriteFile(tmpPath, []byte("binary"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := moveServiceUpgradeFile(tmpPath, targetPath); err != nil {
		t.Fatalf("moveServiceUpgradeFile() error = %v", err)
	}

	info, err := os.Stat(targetPath)
	if err != nil {
		t.Fatalf("Stat(target) error = %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Fatalf("target mode = %v, executable bit is not set", info.Mode())
	}
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Fatalf("tmp path still exists or stat failed unexpectedly: %v", err)
	}
}

func useTempServicesConfig(t *testing.T) {
	t.Helper()
	oldPath := servicesConfigPath
	servicesConfigPath = filepath.Join(t.TempDir(), "services.json")
	t.Cleanup(func() {
		servicesConfigPath = oldPath
	})
}

type testPortForwardProvider struct {
	name string

	mu    sync.Mutex
	stops int
}

func (p *testPortForwardProvider) Name() string        { return p.name }
func (p *testPortForwardProvider) DisplayName() string { return p.name }
func (p *testPortForwardProvider) Description() string { return p.name }
func (p *testPortForwardProvider) Available() bool     { return true }

func (p *testPortForwardProvider) Start(port int, hostname string) (*portforward.TunnelHandle, error) {
	resultCh := make(chan portforward.TunnelResult, 1)
	resultCh <- portforward.TunnelResult{PublicURL: fmt.Sprintf("https://%s", hostname)}
	return &portforward.TunnelHandle{
		Result: resultCh,
		Stop: func() {
			p.mu.Lock()
			p.stops++
			p.mu.Unlock()
		},
		Logs: portforward.NewLogBuffer(),
	}, nil
}

func (p *testPortForwardProvider) StopCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stops
}
