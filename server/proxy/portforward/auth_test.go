package portforward

import (
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/xhd2015/ai-critic/server/config"
)

func TestAddForwardWithAuthTunnelsHopNotBackend(t *testing.T) {
	m := NewManager()
	m.authSecretPath = filepath.Join(t.TempDir(), "secret")
	provider := &testProvider{name: ProviderCloudflareOwned}
	m.RegisterProvider(provider)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	backendPort := ln.Addr().(*net.TCPAddr).Port

	pf, err := m.AddForward(ForwardRequest{
		Port:     backendPort,
		Label:    "auth-app.xhd2015.xyz",
		Provider: ProviderCloudflareOwned,
		Auth: config.PortAuth{
			Required:  true,
			TokenMode: config.PortAuthCustom,
			Token:     "s3cret",
		},
	})
	if err != nil {
		t.Fatalf("AddForward: %v", err)
	}
	t.Cleanup(func() { _ = m.Remove(backendPort) })

	if pf.LocalPort != backendPort {
		t.Fatalf("LocalPort = %d, want backend %d", pf.LocalPort, backendPort)
	}
	if pf.Auth == nil || !pf.Auth.Required {
		t.Fatalf("Auth = %#v", pf.Auth)
	}
	if pf.Auth.Token != "" {
		t.Fatal("list/SSE PortForward must not include token")
	}
	hop := provider.LastPort()
	if hop == 0 || hop == backendPort {
		t.Fatalf("provider Start port = %d, want hop != backend %d", hop, backendPort)
	}

	resp, err := http.Get("http://127.0.0.1:" + strconv.Itoa(hop) + "/")
	if err != nil {
		t.Fatalf("GET hop: %v", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusOK {
		t.Fatalf("unauthenticated hop status = %d", resp.StatusCode)
	}

	forwards := m.List()
	if len(forwards) != 1 || forwards[0].LocalPort != backendPort || forwards[0].Auth == nil || !forwards[0].Auth.Required {
		t.Fatalf("list = %#v", forwards)
	}
}

func TestRemoveClosesAuthHop(t *testing.T) {
	m := NewManager()
	m.authSecretPath = filepath.Join(t.TempDir(), "secret")
	provider := &testProvider{name: ProviderCloudflareOwned}
	m.RegisterProvider(provider)

	pf, err := m.AddForward(ForwardRequest{
		Port:     9476,
		Label:    "auth-app.xhd2015.xyz",
		Provider: ProviderCloudflareOwned,
		Auth: config.PortAuth{
			Required:  true,
			TokenMode: config.PortAuthCustom,
			Token:     "s3cret",
		},
	})
	if err != nil {
		t.Fatalf("AddForward: %v", err)
	}
	if pf.Auth == nil || !pf.Auth.Required {
		t.Fatal("Auth.Required = false")
	}
	hop := provider.LastPort()
	if err := m.Remove(9476); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	_, err = net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(hop))
	if err == nil {
		t.Fatal("auth hop still listening after Remove")
	}
}

func TestAddWithoutAuthStillTunnelsBackendPort(t *testing.T) {
	m := NewManager()
	provider := &testProvider{name: ProviderCloudflareOwned}
	m.RegisterProvider(provider)
	if _, err := m.Add(9476, "plain.xhd2015.xyz", ProviderCloudflareOwned); err != nil {
		t.Fatalf("Add: %v", err)
	}
	t.Cleanup(func() { _ = m.Remove(9476) })
	if provider.LastPort() != 9476 {
		t.Fatalf("Start port = %d, want 9476", provider.LastPort())
	}
}

func TestResolveForwardAuthGeneratesEphemeral(t *testing.T) {
	dir := t.TempDir()
	oldFile := config.PortMappingAuthFile
	oldDir := config.DataDir
	config.DataDir = dir
	config.PortMappingAuthFile = filepath.Join(dir, "port-mapping-auth.json")
	t.Cleanup(func() {
		config.PortMappingAuthFile = oldFile
		config.DataDir = oldDir
	})

	got, err := resolveForwardAuth(8422, &config.PortAuth{Required: true})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Ephemeral || got.TokenMode != config.PortAuthEphemeral || len(got.Token) != 32 {
		t.Fatalf("generated = %#v", got)
	}
	if err := config.SetPortMappingAuth(8422, got); err != nil {
		t.Fatal(err)
	}

	reused, err := resolveForwardAuth(8422, &config.PortAuth{Required: true})
	if err != nil {
		t.Fatal(err)
	}
	if reused.Token != got.Token {
		t.Fatalf("after persist reuse = %q, want %q", reused.Token, got.Token)
	}
}

func TestResolveForwardAuthMigratesShared(t *testing.T) {
	dir := t.TempDir()
	oldFile := config.PortMappingAuthFile
	oldDir := config.DataDir
	config.DataDir = dir
	config.PortMappingAuthFile = filepath.Join(dir, "port-mapping-auth.json")
	t.Cleanup(func() {
		config.PortMappingAuthFile = oldFile
		config.DataDir = oldDir
	})
	if err := config.SetPortMappingAuth(8422, config.PortAuth{Required: true, TokenMode: config.PortAuthShared}); err != nil {
		t.Fatal(err)
	}
	got, err := resolveForwardAuth(8422, &config.PortAuth{Required: true})
	if err != nil {
		t.Fatal(err)
	}
	if got.TokenMode != config.PortAuthEphemeral || got.Token == "" {
		t.Fatalf("migrate shared = %#v", got)
	}
}
