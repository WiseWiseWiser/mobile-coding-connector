package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhd2015/ai-critic/server/auth"
	"github.com/xhd2015/ai-critic/server/config"
	"github.com/xhd2015/ai-critic/server/proxy/portforward"
)

func TestValidateDefinitionRequireAuthNeedsPort(t *testing.T) {
	err := validateDefinition(ServiceDefinition{
		Name:        "web",
		Command:     "run",
		RequireAuth: true,
	})
	if err == nil || !strings.Contains(err.Error(), "port forwarding") {
		t.Fatalf("error = %v, want port forwarding required", err)
	}
}

func TestValidateDefinitionCustomTokenRequired(t *testing.T) {
	err := validateDefinition(ServiceDefinition{
		Name:          "web",
		Command:       "run",
		RequireAuth:   true,
		AuthTokenMode: AuthTokenModeCustom,
		PortForward:   &ServicePortForward{Port: 3000},
	})
	if err == nil || !strings.Contains(err.Error(), "custom auth token") {
		t.Fatalf("error = %v, want custom token required", err)
	}
}

func TestNormalizeAuthFieldsSharedClearsToken(t *testing.T) {
	def := ServiceDefinition{
		RequireAuth:   true,
		AuthTokenMode: AuthTokenModeShared,
		AuthToken:     "leftover",
	}
	normalizeAuthFields(&def)
	if def.AuthTokenMode != AuthTokenModeShared || def.AuthToken != "" {
		t.Fatalf("got mode=%q token=%q, want shared with empty token", def.AuthTokenMode, def.AuthToken)
	}
}

func TestInspectAuthTokensCustomAndShared(t *testing.T) {
	custom := inspectAuthTokens(ServiceDefinition{
		RequireAuth:   true,
		AuthTokenMode: AuthTokenModeCustom,
		AuthToken:     "s3cret",
	})
	if len(custom) != 1 || custom[0] != "s3cret" {
		t.Fatalf("custom tokens = %#v, want [s3cret]", custom)
	}

	credFile := filepath.Join(t.TempDir(), "server-credentials")
	if err := os.WriteFile(credFile, []byte("beta-token\nalpha-token\n"), 0600); err != nil {
		t.Fatal(err)
	}
	auth.SetCredentialsFile(credFile)
	t.Cleanup(func() { auth.SetCredentialsFile(config.CredentialsFile) })

	shared := inspectAuthTokens(ServiceDefinition{
		RequireAuth:   true,
		AuthTokenMode: AuthTokenModeShared,
	})
	if len(shared) != 2 || shared[0] != "alpha-token" || shared[1] != "beta-token" {
		t.Fatalf("shared tokens = %#v, want sorted alpha/beta", shared)
	}
}

func TestDefinitionChangedDetectsAuth(t *testing.T) {
	oldDef := ServiceDefinition{Name: "web", Command: "run"}
	newDef := ServiceDefinition{Name: "web", Command: "run", RequireAuth: true, AuthTokenMode: AuthTokenModeShared}
	if !definitionChanged(oldDef, newDef) {
		t.Fatal("expected auth change to count as definition change")
	}
}

func TestCreateOrUpdatePersistsAuthFields(t *testing.T) {
	useTempServicesConfig(t)

	m := &Manager{
		definitions: []ServiceDefinition{},
		processes:   map[string]*serviceProcess{},
	}
	saved, err := m.CreateOrUpdateNoRestart(ServiceDefinition{
		Name:          "web",
		Command:       "run",
		RequireAuth:   true,
		AuthUser:      "alice",
		AuthTokenMode: AuthTokenModeCustom,
		AuthToken:     "s3cret",
		PortForward:   &ServicePortForward{Port: 3000},
	})
	if err != nil {
		t.Fatalf("CreateOrUpdateNoRestart() error = %v", err)
	}
	if !saved.RequireAuth || saved.AuthUser != "alice" || saved.AuthToken != "s3cret" {
		t.Fatalf("status auth = %#v", saved)
	}
	if len(saved.AuthTokens) != 1 || saved.AuthTokens[0] != "s3cret" {
		t.Fatalf("inspect tokens = %#v, want [s3cret]", saved.AuthTokens)
	}
	if m.definitions[0].AuthToken != "s3cret" || m.definitions[0].AuthTokenMode != AuthTokenModeCustom {
		t.Fatalf("saved definition auth = %#v", m.definitions[0])
	}
}

func TestEnsurePortForwardWithAuthTunnelsProxyPort(t *testing.T) {
	pfm := portforward.NewManager()
	provider := &testPortForwardProvider{name: portforward.ProviderCloudflareOwned}
	pfm.RegisterProvider(provider)

	m := &Manager{
		processes: map[string]*serviceProcess{
			"svc-auth": {},
		},
		portForwardManager: pfm,
		dataDir:            t.TempDir(),
	}

	err := m.ensurePortForward("svc-auth", ServiceDefinition{
		RequireAuth:   true,
		AuthTokenMode: AuthTokenModeCustom,
		AuthToken:     "s3cret",
		PortForward: &ServicePortForward{
			Port:       9476,
			Provider:   portforward.ProviderCloudflareOwned,
			BaseDomain: "xhd2015.xyz",
			Subdomain:  "auth-app",
		},
	})
	if err != nil {
		t.Fatalf("ensurePortForward() error = %v", err)
	}
	t.Cleanup(func() { m.teardownAuthHop("svc-auth") })

	proc := m.processes["svc-auth"]
	if proc.authProxy == nil || !proc.authProxy.Alive() {
		t.Fatal("auth proxy was not started")
	}
	if proc.tunneledPort == 0 || proc.tunneledPort == 9476 {
		t.Fatalf("tunneledPort = %d, want ephemeral proxy port", proc.tunneledPort)
	}

	forwards := pfm.List()
	if len(forwards) != 1 {
		t.Fatalf("forward count = %d, want 1: %#v", len(forwards), forwards)
	}
	if forwards[0].LocalPort != proc.tunneledPort {
		t.Fatalf("forward port = %d, want proxy port %d", forwards[0].LocalPort, proc.tunneledPort)
	}
	if forwards[0].Label != "auth-app.xhd2015.xyz" {
		t.Fatalf("forward label = %q", forwards[0].Label)
	}
}
