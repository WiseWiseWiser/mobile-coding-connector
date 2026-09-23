package services

import (
	"errors"
	"fmt"
	"strings"
)

// ServiceKind separates services the user defines from services the server
// owns. The GUI renders them as "User Services" and "System Services".
type ServiceKind string

const (
	// ServiceKindUser is a definition the user created; the manager runs its
	// command as a child process.
	ServiceKindUser ServiceKind = "user"
	// ServiceKindSystem is a server-owned in-process subsystem. It is never
	// written to services.json, and it cannot be edited, renamed or removed.
	ServiceKindSystem ServiceKind = "system"
)

// SystemRuntime is the live state a system service reports. Only Running is
// required; every other field renders as extra detail when present.
type SystemRuntime struct {
	// Running reports whether the subsystem is currently up.
	Running bool
	// Detail is a short human line, e.g. "listening :21000".
	Detail string
	// PublicURL is the public address the subsystem serves, when it has one.
	PublicURL string
	// Port is the local listen port, when the subsystem listens on one.
	Port int
	// LogPath is a log the GUI can stream, when the subsystem keeps one.
	LogPath string
	// Mocked marks a subsystem whose work is simulated, so the GUI can label it
	// instead of implying a real integration.
	Mocked bool
	// AutoStart mirrors the subsystem's own auto-start switch. Nil means the
	// subsystem has no such switch, and enable/disable is hidden.
	AutoStart *bool
	// Edge is the upstream a proxying subsystem publishes through, when it has
	// one. Rendered as its own line so the address is visible without digging.
	Edge string
	// Hosts lists the public hostnames the subsystem is responsible for, each
	// with its live dial count, so a dead publication is visible per host.
	Hosts []SystemHostStatus
	// Status overrides the running/stopped derivation when set. Used for
	// "error" (a host is dead) and "unknown" (the edge could not be reached).
	Status string
}

// States a published hostname can be in. "starting" means the pool has not
// connected yet within the warm-up grace window; "unknown" means the edge could
// not be reached, so its state could not be established either way.
const (
	SystemHostLive     = "live"
	SystemHostStarting = "starting"
	SystemHostDead     = "dead"
	SystemHostMissing  = "missing"
	SystemHostUnknown  = "unknown"
)

// SystemHostStatus is one public hostname a system service publishes.
type SystemHostStatus struct {
	Host  string `json:"host"`
	Dials int    `json:"dials"`
	State string `json:"state"`
}

// SystemController reports a subsystem's state. It is the only required half:
// a controller that cannot start, stop or reconcile anything is still a valid
// system service, and the GUI renders it as status-only.
type SystemController interface {
	SystemStatus() SystemRuntime
}

// SystemLifecycle is implemented by controllers that can start and stop their
// subsystem. Start and Stop must be safe when already in the target state.
type SystemLifecycle interface {
	StartSystem() error
	StopSystem() error
}

// SystemRestarter is implemented by controllers that can reconcile a running
// subsystem, e.g. by republishing hostnames whose dial pool died.
type SystemRestarter interface {
	RestartSystem() error
}

// SystemHost is one hostname a system service is responsible for publishing.
// ServiceID is the owning user service, and is empty for a domain tunnel.
type SystemHost struct {
	Host      string
	ServiceID string
}

// SystemHostProvider is implemented by controllers that need to know which
// hostnames the manager considers theirs. The manager pushes the set before
// asking for status, because it already holds its lock there: a controller
// that called back into the manager would deadlock.
type SystemHostProvider interface {
	SetSystemHosts(hosts []SystemHost)
}

// Actions names a system service can advertise. The GUI and CLI only offer the
// actions a controller actually implements.
const (
	SystemActionStart   = "start"
	SystemActionStop    = "stop"
	SystemActionRestart = "restart"
)

// systemActions lists the actions a controller supports, in display order.
func systemActions(ctrl SystemController) []string {
	actions := make([]string, 0, 3)
	_, lifecycle := ctrl.(SystemLifecycle)
	_, restarter := ctrl.(SystemRestarter)
	if lifecycle {
		actions = append(actions, SystemActionStart, SystemActionStop)
	}
	if restarter || lifecycle {
		actions = append(actions, SystemActionRestart)
	}
	return actions
}

func systemSupportsAction(ctrl SystemController, action string) bool {
	for _, supported := range systemActions(ctrl) {
		if supported == action {
			return true
		}
	}
	return false
}

// errSystemActionUnsupported explains a refused action in terms of what the
// caller should do instead, rather than just "not supported".
func errSystemActionUnsupported(svc SystemService, action string) error {
	return fmt.Errorf("system service %q does not support %s; %s",
		svc.Name, action, systemActionHint(action))
}

func systemActionHint(action string) string {
	switch action {
	case SystemActionStart, SystemActionStop:
		return "control the individual hostnames in Port Forwarding instead"
	case SystemActionRestart:
		return "it has nothing to reconcile"
	default:
		return "it is owned by the server"
	}
}

// SystemAutoStarter is implemented by controllers that can toggle the
// subsystem's own auto-start flag. A controller that omits it offers no
// enable/disable action.
type SystemAutoStarter interface {
	SetSystemAutoStart(enabled bool) error
}

// SystemService is one server-owned service in a Manager's registry.
type SystemService struct {
	ID          string
	Name        string
	Description string
	Controller  SystemController
}

// Messages reported for system-service enable/disable. They mirror the user
// service wording: the toggle controls boot auto-start, not the live state.
const (
	msgSystemDisable = "The service won't stop immediately unless you manually stop it"
	msgSystemEnable  = "The service won't start immediately until the next server boot"
)

// serviceKindOrUser normalizes a kind read from JSON. An absent kind is a user
// service, which keeps payloads written before kinds existed readable.
func serviceKindOrUser(kind ServiceKind) ServiceKind {
	if kind == ServiceKindSystem {
		return ServiceKindSystem
	}
	return ServiceKindUser
}

// RegisterSystemService adds a server-owned service to this manager. It fails
// when the id is empty, duplicated, or already taken by a user service.
func (m *Manager) RegisterSystemService(svc SystemService) error {
	svc.ID = strings.TrimSpace(svc.ID)
	svc.Name = strings.TrimSpace(svc.Name)
	svc.Description = strings.TrimSpace(svc.Description)
	switch {
	case svc.ID == "":
		return fmt.Errorf("system service id is required")
	case svc.Name == "":
		return fmt.Errorf("system service %s: name is required", svc.ID)
	case svc.Controller == nil:
		return fmt.Errorf("system service %s: controller is required", svc.ID)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.systemServiceLocked(svc.ID); ok {
		return fmt.Errorf("system service %s is already registered", svc.ID)
	}
	if _, ok := m.findDefinitionLocked(svc.ID); ok {
		return fmt.Errorf("system service %s collides with a user service", svc.ID)
	}
	m.systemServices = append(m.systemServices, svc)
	return nil
}

// SystemServices returns the registered system services in registration order.
func (m *Manager) SystemServices() []SystemService {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]SystemService(nil), m.systemServices...)
}

func (m *Manager) systemServiceLocked(id string) (SystemService, bool) {
	for _, svc := range m.systemServices {
		if svc.ID == id {
			return svc, true
		}
	}
	return SystemService{}, false
}

// systemService resolves id to a registered system service. The lock is
// released before returning so callers can run a slow controller method.
func (m *Manager) systemService(id string) (SystemService, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.systemServiceLocked(id)
}

// buildSystemStatusListLocked renders the registry for the API and the GUI.
func (m *Manager) buildSystemStatusListLocked() []ServiceStatus {
	result := make([]ServiceStatus, 0, len(m.systemServices))
	for _, svc := range m.systemServices {
		if provider, ok := svc.Controller.(SystemHostProvider); ok {
			provider.SetSystemHosts(m.cloudflareOwnedHostsLocked())
		}
		runtime := svc.Controller.SystemStatus()

		status := StatusStopped
		if runtime.Status != "" {
			status = runtime.Status
		} else if runtime.Running {
			status = StatusRunning
		}
		enabled := true
		if runtime.AutoStart != nil {
			enabled = *runtime.AutoStart
		}

		result = append(result, ServiceStatus{
			ID:              svc.ID,
			Name:            svc.Name,
			Kind:            ServiceKindSystem,
			Description:     svc.Description,
			Status:          status,
			DesiredRunning:  enabled,
			Enabled:         enabled,
			LogPath:         runtime.LogPath,
			Detail:          runtime.Detail,
			PublicURL:       runtime.PublicURL,
			Port:            runtime.Port,
			Mocked:          runtime.Mocked,
			AutoStartSwitch: runtime.AutoStart != nil,
			Edge:            runtime.Edge,
			Hosts:           runtime.Hosts,
			Actions:         systemActions(svc.Controller),
		})
	}
	return result
}

// setSystemAutoStart toggles a system service's own auto-start switch.
func (m *Manager) setSystemAutoStart(svc SystemService, enabled bool) (*ServiceActionResponse, error) {
	starter, ok := svc.Controller.(SystemAutoStarter)
	if !ok {
		return nil, fmt.Errorf("system service %q has no auto-start switch", svc.Name)
	}
	if err := starter.SetSystemAutoStart(enabled); err != nil {
		return nil, err
	}

	message := msgSystemDisable
	if enabled {
		message = msgSystemEnable
	}
	status, ok := m.statusByID(svc.ID)
	if !ok {
		return nil, fmt.Errorf("system service %s not found after action", svc.ID)
	}
	return &ServiceActionResponse{
		Status:  "ok",
		Message: message,
		Service: status,
	}, nil
}

// ErrSystemServiceImmutable marks a write aimed at a system service. Handlers
// map it to 400 rather than 404: the service exists, it just cannot be changed.
var ErrSystemServiceImmutable = errors.New("system service is owned by the server")

// rejectSystemServiceEdit guards the definition-writing paths. System services
// have no definition to edit, so allowing a write would silently create a user
// service that shadows the registered one.
func (m *Manager) rejectSystemServiceEdit(id string, action string) error {
	svc, ok := m.systemService(id)
	if !ok {
		return nil
	}
	return fmt.Errorf("system service %q cannot be %s: %w", svc.Name, action, ErrSystemServiceImmutable)
}
