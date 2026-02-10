package portforward

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/cloudflare"
	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
)

// PortStatuses defines the possible states
const (
	StatusActive     = "active"
	StatusConnecting = "connecting"
	StatusError      = "error"
	StatusStopped    = "stopped"
)

// Provider names
const (
	ProviderLocaltunnel      = "localtunnel"
	ProviderCloudflareQuick  = "cloudflare_quick"
	ProviderCloudflareTunnel = "cloudflare_tunnel"
	ProviderCloudflareOwned  = "cloudflare_owned"
)

// TunnelResult is sent by providers when the tunnel URL is ready or an error occurs
type TunnelResult struct {
	PublicURL string
	Err       error
}

// TunnelHandle represents a running tunnel that can be stopped
type TunnelHandle struct {
	// Result receives the public URL (or error) when the tunnel is ready.
	// Providers must send exactly one value.
	Result <-chan TunnelResult
	// Stop kills the tunnel process
	Stop func()
	// Logs captures the process output for debugging
	Logs *LogBuffer
}

// Provider is the interface that tunnel providers must implement
type Provider interface {
	// Name returns the provider's identifier (e.g. "localtunnel", "cloudflared")
	Name() string
	// DisplayName returns a human-readable name
	DisplayName() string
	// Description returns a short description of the provider
	Description() string
	// Available returns true if the provider's dependencies are installed
	Available() bool
	// Start begins tunneling the given local port and returns a handle.
	// The hostname parameter is an optional hint for the desired public hostname (used by Cloudflare providers).
	Start(port int, hostname string) (*TunnelHandle, error)
}

// PortForward represents a single port forward entry (API response)
type PortForward struct {
	LocalPort int    `json:"localPort"`
	Label     string `json:"label"`
	PublicURL string `json:"publicUrl"`
	Status    string `json:"status"`
	Provider  string `json:"provider"`
	Error     string `json:"error,omitempty"`
}

// tunnel represents a running tunnel
type tunnel struct {
	port      int
	label     string
	provider  string
	publicURL string
	status    string
	errMsg    string
	stop      func()
	logs      *LogBuffer
}

// Manager manages port forwards using registered providers
type Manager struct {
	mu          sync.Mutex
	tunnels     map[int]*tunnel     // keyed by local port
	providers   map[string]Provider // keyed by provider name
	subscribers map[int]chan []PortForward
	nextSubID   int
}

// NewManager creates a new port forward manager
func NewManager() *Manager {
	return &Manager{
		tunnels:     make(map[int]*tunnel),
		providers:   make(map[string]Provider),
		subscribers: make(map[int]chan []PortForward),
	}
}

// Subscribe returns a channel that receives the full port list on every change.
func (m *Manager) Subscribe() (int, <-chan []PortForward) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := m.nextSubID
	m.nextSubID++
	ch := make(chan []PortForward, 8)
	m.subscribers[id] = ch
	return id, ch
}

// Unsubscribe removes a subscriber.
func (m *Manager) Unsubscribe(id int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ch, ok := m.subscribers[id]; ok {
		close(ch)
		delete(m.subscribers, id)
	}
}

// notifySubscribers sends the current port list to all subscribers.
// Must be called with m.mu held.
func (m *Manager) notifySubscribers() {
	if len(m.subscribers) == 0 {
		return
	}
	ports := m.listLocked()
	for _, ch := range m.subscribers {
		// Non-blocking send — drop if subscriber is slow
		select {
		case ch <- ports:
		default:
		}
	}
}

// listLocked returns the port list. Must be called with m.mu held.
func (m *Manager) listLocked() []PortForward {
	result := make([]PortForward, 0, len(m.tunnels))
	for _, t := range m.tunnels {
		result = append(result, PortForward{
			LocalPort: t.port,
			Label:     t.label,
			PublicURL: t.publicURL,
			Status:    t.status,
			Provider:  t.provider,
			Error:     t.errMsg,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].LocalPort < result[j].LocalPort
	})
	return result
}

// RegisterProvider adds a provider to the manager
func (m *Manager) RegisterProvider(p Provider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.providers[p.Name()] = p
}

// global singleton
var defaultManager = NewManager()

// RegisterProvider registers a provider on the default manager
func RegisterDefaultProvider(p Provider) {
	defaultManager.RegisterProvider(p)
}

// List returns all port forwards, sorted by local port
func (m *Manager) List() []PortForward {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.listLocked()
}

// Add starts a new port forward using the specified provider
func (m *Manager) Add(port int, label string, providerName string) (*PortForward, error) {
	m.mu.Lock()
	if _, exists := m.tunnels[port]; exists {
		m.mu.Unlock()
		return nil, fmt.Errorf("port %d is already being forwarded", port)
	}

	// Default to localtunnel
	if providerName == "" {
		providerName = ProviderLocaltunnel
	}
	p, ok := m.providers[providerName]
	if !ok {
		m.mu.Unlock()
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}

	t := &tunnel{
		port:     port,
		label:    label,
		provider: providerName,
		status:   StatusConnecting,
	}
	m.tunnels[port] = t
	m.notifySubscribers()
	m.mu.Unlock()

	fmt.Printf("[Manager.Add] Starting tunnel with provider: %s, label: %q\n", providerName, label)

	// Start the tunnel
	handle, err := p.Start(port, label)
	if err != nil {
		m.mu.Lock()
		t.status = StatusError
		t.errMsg = err.Error()
		m.notifySubscribers()
		m.mu.Unlock()
		return &PortForward{
			LocalPort: port,
			Label:     label,
			Provider:  providerName,
			Status:    StatusError,
			Error:     err.Error(),
		}, nil
	}

	t.stop = handle.Stop
	t.logs = handle.Logs

	// Wait for result in background
	go func() {
		result := <-handle.Result

		m.mu.Lock()
		defer m.mu.Unlock()
		// Check tunnel still exists (not already removed)
		if _, exists := m.tunnels[port]; !exists {
			return
		}
		if result.Err != nil {
			t.status = StatusError
			t.errMsg = result.Err.Error()
		} else {
			t.status = StatusActive
			t.publicURL = result.PublicURL
		}
		m.notifySubscribers()
	}()

	return &PortForward{
		LocalPort: port,
		Label:     label,
		Provider:  providerName,
		Status:    StatusConnecting,
	}, nil
}

// Remove stops and removes a port forward
func (m *Manager) Remove(port int) error {
	m.mu.Lock()
	t, exists := m.tunnels[port]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("port %d is not being forwarded", port)
	}
	delete(m.tunnels, port)
	m.notifySubscribers()
	m.mu.Unlock()

	if t.stop != nil {
		t.stop()
	}
	return nil
}

// ListProviders returns info about all registered providers
func (m *Manager) ListProviders() []providerInfo {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]providerInfo, 0, len(m.providers))
	for _, p := range m.providers {
		result = append(result, providerInfo{
			ID:          p.Name(),
			Name:        p.DisplayName(),
			Description: p.Description(),
			Available:   p.Available(),
		})
	}
	return result
}

// GetLogs returns log lines for a specific port
func (m *Manager) GetLogs(port int) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	t, exists := m.tunnels[port]
	if !exists {
		return nil, fmt.Errorf("port %d is not being forwarded", port)
	}
	if t.logs == nil {
		return []string{}, nil
	}
	return t.logs.Lines(), nil
}

// LocalPortInfo represents a locally listening port
type LocalPortInfo struct {
	Port    int    `json:"port"`
	PID     int    `json:"pid"`
	PPID    int    `json:"ppid"`
	Command string `json:"command"`
	Cmdline string `json:"cmdline"`
}

// getListeningPorts returns all TCP listening ports with their PIDs, PPIDs and full command lines
func getListeningPorts() ([]LocalPortInfo, error) {
	// Use lsof to get listening TCP ports
	cmd := exec.Command("lsof", "-iTCP", "-sTCP:LISTEN", "-n", "-P")
	output, err := cmd.Output()
	if err != nil {
		// Check if lsof is not found in PATH
		if execErr, ok := err.(*exec.Error); ok && execErr.Err == exec.ErrNotFound {
			return nil, fmt.Errorf("lsof not installed: required to detect local listening ports. Install lsof (macOS: brew install lsof, Linux: apt-get install lsof)")
		}
		return nil, fmt.Errorf("failed to run lsof: %w", err)
	}

	// Collect unique PIDs and basic info from lsof
	type lsofEntry struct {
		port    int
		pid     int
		command string
	}
	var entries []lsofEntry
	pidSet := make(map[int]struct{})

	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}

		// lsof columns: COMMAND PID USER FD TYPE DEVICE SIZE/OFF NODE NAME
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		command := fields[0]
		pid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}

		// Parse NAME column which contains IP:PORT
		nameField := fields[8]
		port := 0
		if idx := strings.LastIndex(nameField, ":"); idx != -1 {
			portStr := nameField[idx+1:]
			port, _ = strconv.Atoi(portStr)
		}

		if port > 0 {
			entries = append(entries, lsofEntry{port: port, pid: pid, command: command})
			pidSet[pid] = struct{}{}
		}
	}

	// Batch-fetch ppid and full cmdline for all PIDs via ps
	ppidMap, cmdlineMap := fetchProcessDetails(pidSet)

	// Build result
	ports := make([]LocalPortInfo, 0, len(entries))
	for _, e := range entries {
		ports = append(ports, LocalPortInfo{
			Port:    e.port,
			PID:     e.pid,
			PPID:    ppidMap[e.pid],
			Command: e.command,
			Cmdline: cmdlineMap[e.pid],
		})
	}

	sort.Slice(ports, func(i, j int) bool {
		return ports[i].Port < ports[j].Port
	})

	return ports, nil
}

// fetchProcessDetails uses ps to get ppid and full command line for a set of PIDs.
func fetchProcessDetails(pidSet map[int]struct{}) (ppidMap map[int]int, cmdlineMap map[int]string) {
	ppidMap = make(map[int]int, len(pidSet))
	cmdlineMap = make(map[int]string, len(pidSet))

	if len(pidSet) == 0 {
		return
	}

	// Build comma-separated PID list
	pidStrs := make([]string, 0, len(pidSet))
	for pid := range pidSet {
		pidStrs = append(pidStrs, strconv.Itoa(pid))
	}

	// ps -o pid=,ppid=,command= -p <pids>
	cmd := exec.Command("ps", "-o", "pid=,ppid=,command=", "-p", strings.Join(pidStrs, ","))
	output, err := cmd.Output()
	if err != nil {
		return
	}

	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: "  PID  PPID COMMAND..."
		// Fields are space-separated, but command may contain spaces
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		ppid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		// Rejoin remaining fields as the full command line
		cmdline := strings.Join(fields[2:], " ")
		ppidMap[pid] = ppid
		cmdlineMap[pid] = cmdline
	}

	return
}

// --- HTTP API ---

// RegisterAPI registers the port forwarding API endpoints
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/ports", handlePorts)
	mux.HandleFunc("/api/ports/events", handlePortEvents)
	mux.HandleFunc("/api/ports/providers", handleProviders)
	mux.HandleFunc("/api/ports/logs", handlePortLogs)
	mux.HandleFunc("/api/ports/diagnostics", handleDiagnostics)
	mux.HandleFunc("/api/ports/local", handleLocalPorts)
	mux.HandleFunc("/api/ports/local/events", handleLocalPortEvents)
}

func handleLocalPorts(w http.ResponseWriter, r *http.Request) {
	ports, err := getListeningPorts()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ports)
}

// handleLocalPortEvents streams local listening ports via SSE, polling every 3 seconds.
func handleLocalPortEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Send initial state
	ports, err := getListeningPorts()
	if err != nil {
		// Send error and close connection for critical errors like missing lsof
		errMsg := err.Error()
		fmt.Fprintf(w, "data: {\"error\":%q,\"fatal\":true}\n\n", errMsg)
		flusher.Flush()
		return
	}
	data, _ := json.Marshal(ports)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()

	// Poll and send updates every 3 seconds until client disconnects
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	prevJSON := string(data)
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			ports, err := getListeningPorts()
			if err != nil {
				continue
			}
			newData, _ := json.Marshal(ports)
			newJSON := string(newData)
			// Only send if data changed
			if newJSON == prevJSON {
				continue
			}
			prevJSON = newJSON
			fmt.Fprintf(w, "data: %s\n\n", newData)
			flusher.Flush()
		}
	}
}

func handlePorts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleListPorts(w, r)
	case http.MethodPost:
		handleAddPort(w, r)
	case http.MethodDelete:
		handleRemovePort(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handlePortEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Subscribe to changes
	subID, ch := defaultManager.Subscribe()
	defer defaultManager.Unsubscribe(subID)

	// Send initial state
	data, _ := json.Marshal(defaultManager.List())
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()

	// Stream updates until client disconnects
	for {
		select {
		case <-r.Context().Done():
			return
		case ports, ok := <-ch:
			if !ok {
				return
			}
			data, _ := json.Marshal(ports)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func handleListPorts(w http.ResponseWriter, _ *http.Request) {
	ports := defaultManager.List()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ports)
}

type addPortRequest struct {
	Port       int    `json:"port"`
	Label      string `json:"label"`
	Provider   string `json:"provider"`
	BaseDomain string `json:"baseDomain"`
	Subdomain  string `json:"subdomain"`
}

func handleAddPort(w http.ResponseWriter, r *http.Request) {
	var req addPortRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Printf("[handleAddPort] ERROR decoding request: %v\n", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	fmt.Printf("[handleAddPort] RECEIVED: port=%d provider=%q label=%q baseDomain=%q subdomain=%q\n",
		req.Port, req.Provider, req.Label, req.BaseDomain, req.Subdomain)

	if req.Port <= 0 || req.Port > 65535 {
		http.Error(w, "invalid port number", http.StatusBadRequest)
		return
	}
	if req.Label == "" {
		req.Label = fmt.Sprintf("Port %d", req.Port)
	}

	// For Cloudflare providers, if subdomain is provided, construct hostname from subdomain + baseDomain
	hostname := req.Label
	isCloudflareProvider := req.Provider == ProviderCloudflareOwned || req.Provider == ProviderCloudflareTunnel
	fmt.Printf("[handleAddPort] isCloudflareProvider=%v (provider=%q, expected=%q or %q)\n",
		isCloudflareProvider, req.Provider, ProviderCloudflareOwned, ProviderCloudflareTunnel)

	if isCloudflareProvider && req.Subdomain != "" {
		fmt.Printf("[handleAddPort] Cloudflare provider with subdomain=%q, baseDomain=%q\n", req.Subdomain, req.BaseDomain)
		if req.BaseDomain != "" {
			hostname = fmt.Sprintf("%s.%s", req.Subdomain, req.BaseDomain)
			fmt.Printf("[handleAddPort] Constructed hostname from subdomain + baseDomain: %s\n", hostname)
		} else {
			// If no base domain provided but we have subdomain, use the first owned domain
			ownedDomains := cloudflare.GetOwnedDomains()
			if len(ownedDomains) > 0 {
				hostname = fmt.Sprintf("%s.%s", req.Subdomain, ownedDomains[0])
				fmt.Printf("[handleAddPort] Constructed hostname using first owned domain: %s\n", hostname)
			} else {
				// Fallback: use label as-is (backward compatibility)
				fmt.Printf("[handleAddPort] No owned domains found, using label: %s\n", hostname)
			}
		}
	} else {
		fmt.Printf("[handleAddPort] Using label as hostname: %s (isCloudflare=%v, hasSubdomain=%v)\n", hostname, isCloudflareProvider, req.Subdomain != "")
	}

	pf, err := defaultManager.Add(req.Port, hostname, req.Provider)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(pf)
}

func handleRemovePort(w http.ResponseWriter, r *http.Request) {
	portStr := r.URL.Query().Get("port")
	if portStr == "" {
		var req struct {
			Port int `json:"port"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil && req.Port > 0 {
			portStr = strconv.Itoa(req.Port)
		}
	}

	if portStr == "" {
		http.Error(w, "port parameter is required", http.StatusBadRequest)
		return
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		http.Error(w, "invalid port number", http.StatusBadRequest)
		return
	}

	if err := defaultManager.Remove(port); err != nil {
		if strings.Contains(err.Error(), "not being forwarded") {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "removed"})
}

type providerInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Available   bool   `json:"available"`
}

func handlePortLogs(w http.ResponseWriter, r *http.Request) {
	portStr := r.URL.Query().Get("port")
	if portStr == "" {
		http.Error(w, "port parameter is required", http.StatusBadRequest)
		return
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		http.Error(w, "invalid port number", http.StatusBadRequest)
		return
	}

	logs, err := defaultManager.GetLogs(port)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

func handleProviders(w http.ResponseWriter, _ *http.Request) {
	providers := defaultManager.ListProviders()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(providers)
}

// --- Diagnostics ---

type diagnosticCheck struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Status      string `json:"status"` // "ok", "warning", "error"
	Description string `json:"description"`
}

type diagnosticsResponse struct {
	Overall string            `json:"overall"` // "ok", "warning", "error"
	Checks  []diagnosticCheck `json:"checks"`
}

func handleDiagnostics(w http.ResponseWriter, _ *http.Request) {
	var checks []diagnosticCheck
	overall := "ok"

	// 1. Check if cloudflare_tunnel provider is configured
	cfgOk := false
	if cfg := config.Get(); cfg != nil {
		for _, p := range cfg.PortForwarding.Providers {
			if p.Type == ProviderCloudflareTunnel && p.IsEnabled() && p.Cloudflare != nil {
				cfgOk = true
				checks = append(checks, diagnosticCheck{
					ID:          "config",
					Label:       "Configuration",
					Status:      "ok",
					Description: fmt.Sprintf("Cloudflare tunnel configured with base_domain: %s", p.Cloudflare.BaseDomain),
				})
				break
			}
		}
	}
	if !cfgOk {
		checks = append(checks, diagnosticCheck{
			ID:          "config",
			Label:       "Configuration",
			Status:      "error",
			Description: "No cloudflare_tunnel provider configured in config. Add a port_forwarding.providers entry with type 'cloudflare_tunnel'.",
		})
		overall = "error"
	}

	// 2. Check if cloudflared is installed
	if IsCommandAvailable("cloudflared") {
		version := ""
		out, err := exec.Command("cloudflared", "--version").CombinedOutput()
		if err == nil {
			version = strings.TrimSpace(string(out))
		}
		checks = append(checks, diagnosticCheck{
			ID:          "installed",
			Label:       "cloudflared installed",
			Status:      "ok",
			Description: version,
		})
	} else {
		checks = append(checks, diagnosticCheck{
			ID:          "installed",
			Label:       "cloudflared installed",
			Status:      "error",
			Description: "cloudflared is not installed. Install it from https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/",
		})
		if overall != "error" {
			overall = "error"
		}
	}

	// 3. Check if authenticated (has tunnel list access)
	if IsCommandAvailable("cloudflared") {
		out, err := exec.Command("cloudflared", "tunnel", "list", "--output", "json").CombinedOutput()
		if err != nil {
			errStr := strings.TrimSpace(string(out))
			if strings.Contains(errStr, "login") || strings.Contains(errStr, "auth") || strings.Contains(errStr, "certificate") {
				checks = append(checks, diagnosticCheck{
					ID:          "authenticated",
					Label:       "Cloudflare authenticated",
					Status:      "error",
					Description: "Not authenticated. Run 'cloudflared tunnel login' to authenticate.",
				})
				if overall != "error" {
					overall = "error"
				}
			} else {
				checks = append(checks, diagnosticCheck{
					ID:          "authenticated",
					Label:       "Cloudflare authenticated",
					Status:      "warning",
					Description: fmt.Sprintf("Could not verify authentication: %s", errStr),
				})
				if overall == "ok" {
					overall = "warning"
				}
			}
		} else {
			checks = append(checks, diagnosticCheck{
				ID:          "authenticated",
				Label:       "Cloudflare authenticated",
				Status:      "ok",
				Description: "Authenticated and can list tunnels.",
			})
		}
	}

	// 4. Check if the tunnel exists
	if cfgOk && IsCommandAvailable("cloudflared") {
		cfg := config.Get()
		for _, p := range cfg.PortForwarding.Providers {
			if p.Type == ProviderCloudflareTunnel && p.Cloudflare != nil {
				tunnelRef := p.Cloudflare.TunnelName
				if tunnelRef == "" {
					tunnelRef = p.Cloudflare.TunnelID
				}
				out, err := exec.Command("cloudflared", "tunnel", "info", tunnelRef).CombinedOutput()
				if err != nil {
					checks = append(checks, diagnosticCheck{
						ID:          "tunnel_exists",
						Label:       fmt.Sprintf("Tunnel '%s'", tunnelRef),
						Status:      "error",
						Description: fmt.Sprintf("Tunnel not found or error: %s", strings.TrimSpace(string(out))),
					})
					if overall != "error" {
						overall = "error"
					}
				} else {
					checks = append(checks, diagnosticCheck{
						ID:          "tunnel_exists",
						Label:       fmt.Sprintf("Tunnel '%s'", tunnelRef),
						Status:      "ok",
						Description: "Tunnel exists and is accessible.",
					})
				}
				break
			}
		}
	}

	// 5. Check credentials file
	if cfgOk {
		cfg := config.Get()
		for _, p := range cfg.PortForwarding.Providers {
			if p.Type == ProviderCloudflareTunnel && p.Cloudflare != nil {
				credFile := p.Cloudflare.CredentialsFile
				if credFile == "" {
					configPath := p.Cloudflare.ConfigPath
					if configPath == "" {
						configPath = "./cloudflared"
					}
					if p.Cloudflare.TunnelID != "" {
						credFile = fmt.Sprintf("%s/%s.json", configPath, p.Cloudflare.TunnelID)
					}
				}

				if credFile != "" {
					if _, err := os.Stat(credFile); err != nil {
						checks = append(checks, diagnosticCheck{
							ID:          "credentials",
							Label:       "Credentials file",
							Status:      "error",
							Description: fmt.Sprintf("Credentials file not found: %s", credFile),
						})
						if overall != "error" {
							overall = "error"
						}
					} else {
						checks = append(checks, diagnosticCheck{
							ID:          "credentials",
							Label:       "Credentials file",
							Status:      "ok",
							Description: fmt.Sprintf("Found: %s", credFile),
						})
					}
				}
				break
			}
		}
	}

	resp := diagnosticsResponse{
		Overall: overall,
		Checks:  checks,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
