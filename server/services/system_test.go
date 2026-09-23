package services

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// fakeSystemController records the lifecycle calls a system service receives.
type fakeSystemController struct {
	runtime      SystemRuntime
	started      int
	stopped      int
	autoStartSet []bool
	startErr     error
	stopErr      error
	autoStartErr error
}

func (f *fakeSystemController) SystemStatus() SystemRuntime { return f.runtime }

func (f *fakeSystemController) StartSystem() error {
	f.started++
	return f.startErr
}

func (f *fakeSystemController) StopSystem() error {
	f.stopped++
	return f.stopErr
}

func (f *fakeSystemController) SetSystemAutoStart(enabled bool) error {
	if f.autoStartErr != nil {
		return f.autoStartErr
	}
	f.autoStartSet = append(f.autoStartSet, enabled)
	return nil
}

// noAutoStartController has no auto-start switch, so it must not be offered
// enable/disable.
type noAutoStartController struct{ runtime SystemRuntime }

func (c *noAutoStartController) SystemStatus() SystemRuntime { return c.runtime }
func (c *noAutoStartController) StartSystem() error          { return nil }
func (c *noAutoStartController) StopSystem() error           { return nil }

func boolRef(v bool) *bool { return &v }

// newSystemManager returns a Manager holding one user service and one system
// service backed by ctrl, isolated from the real services.json.
func newSystemManager(t *testing.T, ctrl SystemController) *Manager {
	t.Helper()
	useTempServicesConfig(t)
	m := NewManagerFromDefinitions([]ServiceDefinition{
		{ID: "svc-user", Name: "web", Command: "run"},
	})
	if err := m.RegisterSystemService(SystemService{
		ID:          "sys-fake",
		Name:        "Fake System",
		Description: "test subsystem",
		Controller:  ctrl,
	}); err != nil {
		t.Fatalf("RegisterSystemService() error = %v", err)
	}
	return m
}

func statusByID(t *testing.T, m *Manager, id string) ServiceStatus {
	t.Helper()
	for _, status := range m.List() {
		if status.ID == id {
			return status
		}
	}
	t.Fatalf("service %s not found in %#v", id, m.List())
	return ServiceStatus{}
}

func TestListReturnsUserServicesBeforeSystemServices(t *testing.T) {
	m := newSystemManager(t, &fakeSystemController{})

	list := m.List()
	if len(list) != 2 {
		t.Fatalf("len(List()) = %d, want 2: %#v", len(list), list)
	}
	if list[0].ID != "svc-user" || list[0].Kind != ServiceKindUser {
		t.Fatalf("first entry = %#v, want the user service", list[0])
	}
	if list[1].ID != "sys-fake" || list[1].Kind != ServiceKindSystem {
		t.Fatalf("second entry = %#v, want the system service", list[1])
	}
}

func TestSystemServiceStatusCarriesRuntime(t *testing.T) {
	ctrl := &fakeSystemController{runtime: SystemRuntime{
		Running:   true,
		Detail:    "listening :21000",
		PublicURL: "https://sys.example",
		Port:      21000,
		Mocked:    true,
		AutoStart: boolRef(false),
	}}
	m := newSystemManager(t, ctrl)

	status := statusByID(t, m, "sys-fake")
	if status.Kind != ServiceKindSystem {
		t.Fatalf("Kind = %q, want system", status.Kind)
	}
	if status.Status != StatusRunning {
		t.Fatalf("Status = %q, want running", status.Status)
	}
	if status.Detail != "listening :21000" || status.PublicURL != "https://sys.example" || status.Port != 21000 {
		t.Fatalf("runtime detail = %#v", status)
	}
	if !status.Mocked {
		t.Fatalf("Mocked = false, want true")
	}
	if status.Enabled || status.DesiredRunning {
		t.Fatalf("Enabled = %v, want false from the subsystem auto-start switch", status.Enabled)
	}
	if status.Description != "test subsystem" {
		t.Fatalf("Description = %q", status.Description)
	}
	// A system service has no command and no definition-owned fields.
	if status.Command != "" || status.PID != 0 {
		t.Fatalf("system status leaked command fields: %#v", status)
	}
}

func TestStartDispatchesToSystemController(t *testing.T) {
	ctrl := &fakeSystemController{runtime: SystemRuntime{AutoStart: boolRef(true)}}
	m := newSystemManager(t, ctrl)

	status, err := m.Start("sys-fake")
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if ctrl.started != 1 {
		t.Fatalf("controller started %d times, want 1", ctrl.started)
	}
	if status == nil || status.ID != "sys-fake" {
		t.Fatalf("Start() status = %#v", status)
	}
	// No process may be spawned for a system service.
	if proc := m.processes["sys-fake"]; proc != nil {
		t.Fatalf("system service spawned a process: %#v", proc)
	}
}

func TestStopDispatchesToSystemController(t *testing.T) {
	ctrl := &fakeSystemController{runtime: SystemRuntime{Running: true}}
	m := newSystemManager(t, ctrl)

	if err := m.Stop("sys-fake"); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if ctrl.stopped != 1 {
		t.Fatalf("controller stopped %d times, want 1", ctrl.stopped)
	}
}

func TestRestartStopsThenStartsSystemController(t *testing.T) {
	ctrl := &fakeSystemController{runtime: SystemRuntime{AutoStart: boolRef(true)}}
	m := newSystemManager(t, ctrl)

	if err := m.Restart("sys-fake"); err != nil {
		t.Fatalf("Restart() error = %v", err)
	}
	if ctrl.stopped != 1 || ctrl.started != 1 {
		t.Fatalf("restart calls: stopped=%d started=%d, want 1 and 1", ctrl.stopped, ctrl.started)
	}
}

func TestStartPropagatesSystemControllerError(t *testing.T) {
	ctrl := &fakeSystemController{startErr: errors.New("tunnel unavailable")}
	m := newSystemManager(t, ctrl)

	if _, err := m.Start("sys-fake"); err == nil || !strings.Contains(err.Error(), "tunnel unavailable") {
		t.Fatalf("Start() error = %v, want the controller error", err)
	}
}

func TestDeleteRejectsSystemServiceBeforeStoppingIt(t *testing.T) {
	ctrl := &fakeSystemController{runtime: SystemRuntime{Running: true}}
	m := newSystemManager(t, ctrl)

	err := m.Delete("sys-fake")
	if !errors.Is(err, ErrSystemServiceImmutable) {
		t.Fatalf("Delete() error = %v, want ErrSystemServiceImmutable", err)
	}
	if ctrl.stopped != 0 {
		t.Fatalf("rejected delete stopped the subsystem %d times, want 0", ctrl.stopped)
	}
	if got := len(m.List()); got != 2 {
		t.Fatalf("len(List()) = %d, want the system service still registered", got)
	}
}

func TestCreateOrUpdateRejectsSystemServiceID(t *testing.T) {
	m := newSystemManager(t, &fakeSystemController{})

	_, err := m.CreateOrUpdateNoRestart(ServiceDefinition{
		ID:      "sys-fake",
		Name:    "shadow",
		Command: "echo shadow",
	})
	if !errors.Is(err, ErrSystemServiceImmutable) {
		t.Fatalf("createOrUpdate error = %v, want ErrSystemServiceImmutable", err)
	}
	if got := len(m.definitions); got != 1 {
		t.Fatalf("definitions = %d, want the user service only", got)
	}
}

func TestUpgradeRejectsSystemServiceID(t *testing.T) {
	m := newSystemManager(t, &fakeSystemController{})

	_, err := m.Upgrade(ServiceUpgradeRequest{ID: "sys-fake", TmpPath: "/tmp/x"})
	if !errors.Is(err, ErrSystemServiceImmutable) {
		t.Fatalf("Upgrade() error = %v, want ErrSystemServiceImmutable", err)
	}
}

func TestDisableEnableTogglesSystemAutoStart(t *testing.T) {
	ctrl := &fakeSystemController{runtime: SystemRuntime{AutoStart: boolRef(true)}}
	m := newSystemManager(t, ctrl)

	response, err := m.Disable("sys-fake")
	if err != nil {
		t.Fatalf("Disable() error = %v", err)
	}
	if response.Message != msgSystemDisable {
		t.Fatalf("disable message = %q", response.Message)
	}
	if response.Service == nil || response.Service.ID != "sys-fake" {
		t.Fatalf("disable response service = %#v", response.Service)
	}

	if _, err := m.Enable("sys-fake"); err != nil {
		t.Fatalf("Enable() error = %v", err)
	}
	want := []bool{false, true}
	if len(ctrl.autoStartSet) != len(want) {
		t.Fatalf("auto-start calls = %v, want %v", ctrl.autoStartSet, want)
	}
	for i, v := range want {
		if ctrl.autoStartSet[i] != v {
			t.Fatalf("auto-start calls = %v, want %v", ctrl.autoStartSet, want)
		}
	}
	// The toggle must not start or stop the subsystem.
	if ctrl.started != 0 || ctrl.stopped != 0 {
		t.Fatalf("enable/disable touched the lifecycle: started=%d stopped=%d", ctrl.started, ctrl.stopped)
	}
}

func TestDisableRejectsControllerWithoutAutoStart(t *testing.T) {
	m := newSystemManager(t, &noAutoStartController{})

	_, err := m.Disable("sys-fake")
	if err == nil || !strings.Contains(err.Error(), "auto-start") {
		t.Fatalf("Disable() error = %v, want an auto-start error", err)
	}
}

func TestRegisterSystemServiceValidation(t *testing.T) {
	useTempServicesConfig(t)
	m := NewManagerFromDefinitions([]ServiceDefinition{{ID: "svc-user", Name: "web", Command: "run"}})

	cases := []struct {
		name    string
		svc     SystemService
		wantErr string
	}{
		{
			name:    "empty id",
			svc:     SystemService{Name: "x", Controller: &fakeSystemController{}},
			wantErr: "id is required",
		},
		{
			name:    "empty name",
			svc:     SystemService{ID: "sys-x", Controller: &fakeSystemController{}},
			wantErr: "name is required",
		},
		{
			name:    "nil controller",
			svc:     SystemService{ID: "sys-x", Name: "x"},
			wantErr: "controller is required",
		},
		{
			name:    "collides with a user service",
			svc:     SystemService{ID: "svc-user", Name: "x", Controller: &fakeSystemController{}},
			wantErr: "collides with a user service",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := m.RegisterSystemService(tc.svc)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("RegisterSystemService() error = %v, want %q", err, tc.wantErr)
			}
		})
	}

	if err := m.RegisterSystemService(SystemService{ID: "sys-x", Name: "x", Controller: &fakeSystemController{}}); err != nil {
		t.Fatalf("first registration error = %v", err)
	}
	if err := m.RegisterSystemService(SystemService{ID: "sys-x", Name: "x", Controller: &fakeSystemController{}}); err == nil {
		t.Fatal("duplicate registration must fail")
	}
}

func TestSystemServiceIsNeverPersisted(t *testing.T) {
	ctrl := &fakeSystemController{runtime: SystemRuntime{AutoStart: boolRef(true)}}
	m := newSystemManager(t, ctrl)

	if _, err := m.Start("sys-fake"); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := m.Stop("sys-fake"); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	m.List()

	// A user-service write persists only that user service.
	if _, err := m.CreateOrUpdateNoRestart(ServiceDefinition{ID: "svc-2", Name: "api", Command: "run"}); err != nil {
		t.Fatalf("CreateOrUpdateNoRestart() error = %v", err)
	}

	data, err := os.ReadFile(m.servicesFile())
	if err != nil {
		t.Fatalf("read services.json: %v", err)
	}
	if strings.Contains(string(data), "sys-fake") {
		t.Fatalf("services.json persisted a system service:\n%s", data)
	}
	if !strings.Contains(string(data), "svc-2") {
		t.Fatalf("services.json missing the user service:\n%s", data)
	}
}

func TestSystemServiceHTTPActions(t *testing.T) {
	ctrl := &fakeSystemController{runtime: SystemRuntime{Running: true, AutoStart: boolRef(true)}}
	m := newSystemManager(t, ctrl)

	mux := http.NewServeMux()
	RegisterAPIWithManager(mux, m)
	server := httptest.NewServer(mux)
	defer server.Close()

	post := func(path string) *http.Response {
		t.Helper()
		resp, err := http.Post(server.URL+path, "application/json", nil)
		if err != nil {
			t.Fatalf("POST %s: %v", path, err)
		}
		t.Cleanup(func() { _ = resp.Body.Close() })
		return resp
	}

	if resp := post("/api/services/stop?id=sys-fake"); resp.StatusCode != http.StatusOK {
		t.Fatalf("stop status = %d, want 200", resp.StatusCode)
	}
	if ctrl.stopped != 1 {
		t.Fatalf("HTTP stop reached the controller %d times, want 1", ctrl.stopped)
	}
	if resp := post("/api/services/start?id=sys-fake"); resp.StatusCode != http.StatusOK {
		t.Fatalf("start status = %d, want 200", resp.StatusCode)
	}
	if ctrl.started != 1 {
		t.Fatalf("HTTP start reached the controller %d times, want 1", ctrl.started)
	}

	// Removing a system service is a client error, not a missing resource.
	req, err := http.NewRequest(http.MethodDelete, server.URL+"/api/services?id=sys-fake", nil)
	if err != nil {
		t.Fatalf("new DELETE request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("delete status = %d, want 400", resp.StatusCode)
	}

	// An unknown id still reports 404.
	unknown, err := http.NewRequest(http.MethodDelete, server.URL+"/api/services?id=svc-missing", nil)
	if err != nil {
		t.Fatalf("new DELETE request: %v", err)
	}
	unknownResp, err := http.DefaultClient.Do(unknown)
	if err != nil {
		t.Fatalf("DELETE unknown: %v", err)
	}
	defer unknownResp.Body.Close()
	if unknownResp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown delete status = %d, want 404", unknownResp.StatusCode)
	}
}

func TestListPayloadMarksKinds(t *testing.T) {
	ctrl := &fakeSystemController{runtime: SystemRuntime{AutoStart: boolRef(true)}}
	m := newSystemManager(t, ctrl)

	encoded, err := json.Marshal(m.List())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	payload := string(encoded)
	if !strings.Contains(payload, `"kind":"user"`) {
		t.Fatalf("payload missing user kind:\n%s", payload)
	}
	if !strings.Contains(payload, `"kind":"system"`) {
		t.Fatalf("payload missing system kind:\n%s", payload)
	}
}

func TestPresetsCoverShippedProxyBinaries(t *testing.T) {
	presets := Presets()
	if len(presets) == 0 {
		t.Fatal("Presets() is empty")
	}
	seen := make(map[string]bool)
	for _, preset := range presets {
		if preset.ID == "" || preset.Name == "" || preset.Command == "" {
			t.Fatalf("preset missing required fields: %#v", preset)
		}
		if seen[preset.ID] {
			t.Fatalf("duplicate preset id %q", preset.ID)
		}
		seen[preset.ID] = true
	}
	for _, want := range []string{"forward-proxy", "basic-auth-proxy"} {
		if !seen[want] {
			t.Fatalf("preset %q missing from %#v", want, presets)
		}
	}
}
