package wsproxy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/xhd2015/ai-critic/server/config"
)

// BackendType selects the xray outbound for an instance's egress.
//
//	self  — xray "freedom" outbound: the ai-critic host itself is the proxy;
//	        traffic egresses straight into the host's network (intranet).
//	proxy — xray "http" outbound chained to an upstream HTTP proxy, which may
//	        be a local port (127.0.0.1:23456, e.g. the HTTP Proxy system
//	        service) or a remote host (http://<mac-ip>:23456 from an RSP).
type BackendType string

const (
	BackendSelf  BackendType = "self"
	BackendProxy BackendType = "proxy"
)

const instancesFileName = "ws-proxy-instances.json"

// Instance is one independently published WS Proxy endpoint.
type Instance struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	BackendType BackendType `json:"backend_type"`
	BackendURL  string      `json:"backend_url,omitempty"`
	Subdomain   string      `json:"subdomain"`
	// Hostname pins the published domain. Set by migration so an upgraded
	// server keeps the URL phones already have; new instances derive
	// <subdomain>.<domain> at start.
	Hostname   string `json:"hostname,omitempty"`
	WSPath     string `json:"ws_path"`
	UUID       string `json:"uuid"`
	ListenPort int    `json:"listen_port"`
	AutoStart  bool   `json:"auto_start"`
	Enabled    bool   `json:"enabled"`
	// PublicURL is runtime state: the domain this instance last published.
	PublicURL string `json:"public_url,omitempty"`
	// Legacy marks the instance migrated from the single-instance config.
	Legacy bool `json:"legacy,omitempty"`
}

func instancesPath() string {
	if _testConfigDir != "" {
		return filepath.Join(_testConfigDir, instancesFileName)
	}
	return filepath.Join(config.DataDir, instancesFileName)
}

// LoadInstances returns the stored instances, migrating the legacy single-
// instance config into a "default" instance on first read. The legacy config
// file is left in place as a backup; it stops being authoritative after this.
func LoadInstances() ([]Instance, error) {
	if data, err := os.ReadFile(instancesPath()); err == nil {
		var out []Instance
		if err := json.Unmarshal(data, &out); err != nil {
			return nil, fmt.Errorf("failed to parse instances: %w", err)
		}
		return out, nil
	}
	// First run with the instances store: migrate legacy config.
	instances := migrateLegacy()
	if err := SaveInstances(instances); err != nil {
		return nil, err
	}
	return instances, nil
}

// migrateLegacy converts ws-proxy.json into one instance. backend=proxy when
// an upstream was set, otherwise self — matching what the old runtime did.
// A config without a UUID was never started (pure defaults): it migrates to
// an EMPTY store so the controller keeps reporting "not configured" instead
// of fabricating a phantom instance.
func migrateLegacy() []Instance {
	cfg, err := LoadConfig()
	if err != nil || cfg == nil || cfg.UUID == "" {
		return nil
	}
	backend := BackendSelf
	backendURL := ""
	if cfg.UpstreamProxy != "" {
		backend = BackendProxy
		backendURL = cfg.UpstreamProxy
	}
	inst := Instance{
		ID:          "default",
		Name:        "default",
		BackendType: backend,
		BackendURL:  backendURL,
		Subdomain:   cfg.Subdomain,
		Hostname:    HostFromPublicURL(cfg.PublicURL),
		WSPath:      cfg.WSPath,
		UUID:        cfg.UUID,
		ListenPort:  cfg.ListenPort,
		AutoStart:   cfg.AutoStart,
		Enabled:     true,
		PublicURL:   cfg.PublicURL,
		Legacy:      true,
	}
	return []Instance{inst}
}

func SaveInstances(instances []Instance) error {
	path := instancesPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}
	data, err := json.MarshalIndent(instances, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0600)
}

// FindInstance locates an instance by id.
func FindInstance(instances []Instance, id string) (*Instance, int) {
	for i := range instances {
		if instances[i].ID == id {
			return &instances[i], i
		}
	}
	return nil, -1
}

var instanceIDRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,30}$`)
var subdomainRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,30}[a-z0-9])?$`)

// ValidateInstance checks user-supplied fields; uuid/port/ws_path are filled
// from defaults when empty.
func ValidateInstance(inst *Instance) error {
	inst.ID = strings.TrimSpace(inst.ID)
	inst.Name = strings.TrimSpace(inst.Name)
	inst.Subdomain = strings.ToLower(strings.TrimSpace(inst.Subdomain))
	switch inst.BackendType {
	case BackendSelf:
		inst.BackendURL = ""
	case BackendProxy:
		inst.BackendURL = strings.TrimSpace(inst.BackendURL)
		if !strings.Contains(inst.BackendURL, "://") {
			inst.BackendURL = "http://" + inst.BackendURL
		}
		if extractHost(inst.BackendURL) == "" || extractPort(inst.BackendURL) <= 0 {
			return fmt.Errorf("backend url must be http://<host>:<port>")
		}
	default:
		return fmt.Errorf("backend_type must be self or proxy")
	}
	if inst.ID == "" {
		inst.ID = inst.Subdomain
	}
	if !instanceIDRe.MatchString(inst.ID) {
		return fmt.Errorf("id must match %s", instanceIDRe)
	}
	if inst.Name == "" {
		inst.Name = inst.ID
	}
	if !subdomainRe.MatchString(inst.Subdomain) {
		return fmt.Errorf("subdomain must match %s", subdomainRe)
	}
	if inst.WSPath == "" {
		inst.WSPath = defaultWSPath
	}
	if !strings.HasPrefix(inst.WSPath, "/") {
		inst.WSPath = "/" + inst.WSPath
	}
	if inst.UUID == "" {
		inst.UUID = generateUUID()
	}
	return nil
}

// newInstance builds an instance with generated identity fields.
func newInstance(id string, backend BackendType, backendURL string) *Instance {
	return &Instance{
		ID:          id,
		Name:        id,
		BackendType: backend,
		BackendURL:  backendURL,
		Subdomain:   id,
		WSPath:      defaultWSPath,
		UUID:        generateUUID(),
		Enabled:     true,
	}
}

// toConfig renders the instance as the per-instance xray/runtime Config view
// the existing helpers operate on.
func (inst *Instance) toConfig() *Config {
	return &Config{
		UpstreamProxy: inst.BackendURL,
		ListenPort:    inst.ListenPort,
		WSPath:        inst.WSPath,
		UUID:          inst.UUID,
		Subdomain:     inst.Subdomain,
		InstanceID:    inst.ID,
		AutoStart:     inst.AutoStart,
		PublicURL:     inst.PublicURL,
	}
}

// HostnameFor resolves the published hostname: pinned override first (set by
// migration so upgraded phones keep working), else <subdomain>.<domain>.
func (inst *Instance) HostnameFor(domain string) string {
	if inst.Hostname != "" {
		return inst.Hostname
	}
	return inst.Subdomain + "." + domain
}
