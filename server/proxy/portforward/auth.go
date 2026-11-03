package portforward

import (
	"fmt"
	"strings"

	"github.com/xhd2015/ai-critic/server/config"
	"github.com/xhd2015/ai-critic/server/proxy/origin"
	"github.com/xhd2015/ai-critic/server/quicktest"
	"github.com/xhd2015/ai-critic/server/services/authproxy"
)

// ForwardRequest is the input to AddForward.
type ForwardRequest struct {
	ID       string
	Target   *config.OriginTarget
	Port     int
	Label    string
	Provider string
	Auth     config.PortAuth
}

// Add starts a new port forward using the specified provider.
func (m *Manager) Add(port int, label string, providerName string) (*PortForward, error) {
	return m.AddForward(ForwardRequest{Port: port, Label: label, Provider: providerName})
}

func (m *Manager) secretPath() string {
	if m != nil && strings.TrimSpace(m.authSecretPath) != "" {
		return m.authSecretPath
	}
	return config.PortMappingAuthSecretFile
}

func (t *tunnel) asPortForward(includeToken bool) PortForward {
	pf := PortForward{
		ID:        t.id,
		Target:    &t.target,
		LocalPort: t.port,
		Label:     t.label,
		PublicURL: t.publicURL,
		Status:    t.status,
		Provider:  t.provider,
		Error:     t.errMsg,
		Type:      PortForwardTypePortForward,
	}
	if t.auth.Required {
		a := t.auth
		if !includeToken {
			a.Token = ""
		}
		pf.Auth = &a
	}
	return pf
}

func hopTokenMode(auth config.PortAuth) (mode, custom string) {
	if auth.TokenMode == config.PortAuthShared {
		return authproxy.TokenModeShared, ""
	}
	return authproxy.TokenModeCustom, auth.Token
}

// resolveForwardAuth fills in an ephemeral token when require-auth is set
// without a token and without --shared-token. Saved ephemeral/custom tokens
// are reused; leftover shared rows are migrated off server-credentials.
func resolveForwardAuth(port int, req *config.PortAuth) (config.PortAuth, error) {
	if req == nil || !req.Required {
		return config.PortAuth{}, nil
	}
	out := config.PortAuth{
		Required: true,
		User:     strings.TrimSpace(req.User),
	}
	mode := strings.ToLower(strings.TrimSpace(req.TokenMode))
	token := strings.TrimSpace(req.Token)
	if mode == config.PortAuthShared {
		out.TokenMode = config.PortAuthShared
		return out, nil
	}
	if token != "" {
		if mode == config.PortAuthEphemeral {
			out.TokenMode = config.PortAuthEphemeral
			out.Ephemeral = true
		} else {
			out.TokenMode = config.PortAuthCustom
		}
		out.Token = token
		return out, nil
	}
	saved, err := config.GetPortMappingAuth(port)
	if err != nil {
		return config.PortAuth{}, err
	}
	if saved != nil && saved.Token != "" && saved.TokenMode != config.PortAuthShared {
		out.Token = saved.Token
		out.TokenMode = saved.TokenMode
		out.Ephemeral = saved.Ephemeral || saved.TokenMode == config.PortAuthEphemeral
		if out.User == "" {
			out.User = saved.User
		}
		if out.TokenMode == "" {
			if out.Ephemeral {
				out.TokenMode = config.PortAuthEphemeral
			} else {
				out.TokenMode = config.PortAuthCustom
			}
		}
		return out, nil
	}
	generated, err := config.GenerateEphemeralToken()
	if err != nil {
		return config.PortAuth{}, err
	}
	out.Token = generated
	out.TokenMode = config.PortAuthEphemeral
	out.Ephemeral = true
	return out, nil
}

// AddForward publishes a port. When Auth.Required is set, Cloudflare (or the
// chosen provider) is aimed at an in-process login hop, not the backend port.
// Mappings use stable IDs; legacy port selectors must be unambiguous.
func (m *Manager) AddForward(req ForwardRequest) (*PortForward, error) {
	target := config.OriginTarget{}
	if req.Target != nil {
		target = *req.Target
	}
	target, err := target.Normalize(req.Port)
	if err != nil {
		return nil, err
	}
	port := target.Port
	id := req.ID
	if id == "" {
		if target.Host == "127.0.0.1" && target.Scheme == "http" {
			id = config.LocalMappingID(port)
		} else {
			id, err = config.NewMappingID()
			if err != nil {
				return nil, err
			}
		}
	}
	label := req.Label
	providerName := req.Provider

	var proxy *authproxy.Server
	originPort := port
	if req.Auth.Required || req.Target != nil {
		mode, custom := hopTokenMode(req.Auth)
		transport := origin.NewTransport()
		started, err := authproxy.Start(authproxy.Config{
			BackendPort: port,
			BackendURL:  target.URL(),
			DisableAuth: !req.Auth.Required,
			Transport:   transport,
			AuthUser:    req.Auth.User,
			TokenMode:   mode,
			CustomToken: custom,
			SecretPath:  m.secretPath(),
		})
		if err != nil {
			_ = transport.Close()
			return nil, fmt.Errorf("start origin proxy: %w", err)
		}
		transport.Forbid(fmt.Sprintf("127.0.0.1:%d", started.Port()))
		proxy = started
		originPort = started.Port()
	}

	m.mu.Lock()
	if _, exists := m.tunnels[id]; exists {
		m.mu.Unlock()
		if proxy != nil {
			_ = proxy.Close()
		}
		return nil, fmt.Errorf("port %d is already being forwarded", port)
	}

	if providerName == "" {
		providerName = ProviderLocaltunnel
	}
	p, ok := m.providers[providerName]
	if !ok {
		m.mu.Unlock()
		if proxy != nil {
			_ = proxy.Close()
		}
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}

	var staleSameHost []*tunnel
	if isCloudflareHostnameProvider(providerName) {
		for existingPort, existing := range m.tunnels {
			if existingPort == id || !isCloudflareHostnameProvider(existing.provider) {
				continue
			}
			if sameForwardHostname(existing.label, label) {
				fmt.Printf("[Manager.Add] Replacing stale Cloudflare hostname mapping: mapping=%s label=%q -> port=%d\n",
					existingPort, existing.label, port)
				delete(m.tunnels, existingPort)
				staleSameHost = append(staleSameHost, existing)
			}
		}
	}

	t := &tunnel{
		id:         id,
		target:     target,
		port:       port,
		originPort: originPort,
		label:      label,
		provider:   providerName,
		status:     StatusConnecting,
		auth:       req.Auth,
	}
	if proxy != nil {
		t.closeAuth = func() { _ = proxy.Close() }
		t.hopAlive = proxy.Alive
	}
	m.tunnels[id] = t
	m.notifySubscribers()
	m.mu.Unlock()

	for _, stale := range staleSameHost {
		stopTunnel(stale)
	}

	fmt.Printf("[Manager.Add] Starting tunnel with provider: %s, label: %q originPort=%d auth=%v\n", providerName, label, originPort, req.Auth.Required)
	quicktest.LogHeavyOperationWithCallerStack("[Manager.Add] provider=%s label=%q", providerName, label)

	handle, err := p.Start(originPort, label)
	if err != nil {
		m.mu.Lock()
		t.status = StatusError
		t.errMsg = err.Error()
		m.notifySubscribers()
		m.mu.Unlock()
		pf := m.snapshot(t)
		return &pf, nil
	}

	m.mu.Lock()
	if m.tunnels[id] != t {
		m.mu.Unlock()
		if handle.Stop != nil {
			handle.Stop()
		}
		return nil, fmt.Errorf("mapping stopped during tunnel startup")
	}
	t.stop = handle.Stop
	t.logs = handle.Logs
	m.mu.Unlock()

	applyResult := func(result TunnelResult) {
		m.mu.Lock()
		defer m.mu.Unlock()
		if current := m.tunnels[id]; current != t {
			return
		}
		if result.Err != nil {
			t.status = StatusError
			t.errMsg = result.Err.Error()
			t.publicURL = ""
		} else {
			t.status = StatusActive
			t.publicURL = result.PublicURL
			t.errMsg = ""
		}
		m.notifySubscribers()
	}

	select {
	case result := <-handle.Result:
		applyResult(result)
		pf := m.snapshot(t)
		return &pf, nil
	default:
	}

	go func() {
		applyResult(<-handle.Result)
	}()

	pf := m.snapshot(t)
	return &pf, nil
}

func stopTunnel(t *tunnel) {
	if t == nil {
		return
	}
	if t.stop != nil {
		t.stop()
	}
	if t.closeAuth != nil {
		t.closeAuth()
	}
}

func (m *Manager) snapshot(t *tunnel) PortForward {
	m.mu.Lock()
	defer m.mu.Unlock()
	return t.asPortForward(false)
}
