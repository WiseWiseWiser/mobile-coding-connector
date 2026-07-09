package menubar

import "fmt"

const (
	msgDisableRunning = "The server won't stop immediately unless you manually stop it"
	msgEnableStopped  = "The server won't start immediately until daemon checks at next time"
)

// FormatServiceTitle renders the per-service submenu title from status fields.
func FormatServiceTitle(name, status string, enabled bool) string {
	switch status {
	case "running":
		return fmt.Sprintf("%s ● Running", name)
	case "error":
		return fmt.Sprintf("%s ⚠ Error", name)
	case "stopped":
		if !enabled {
			return fmt.Sprintf("%s ○ Stopped (disabled)", name)
		}
		return fmt.Sprintf("%s ○ Stopped", name)
	case "starting":
		return fmt.Sprintf("%s ○ Starting", name)
	default:
		if !enabled {
			return fmt.Sprintf("%s ○ Stopped (disabled)", name)
		}
		return fmt.Sprintf("%s ○ %s", name, status)
	}
}

// CanStopService reports whether the Stop action should be enabled.
func CanStopService(pid int, desiredRunning bool) bool {
	if pid > 0 {
		return true
	}
	return desiredRunning
}

// ShowEnableAction reports whether the menu should offer Enable instead of Disable.
func ShowEnableAction(enabled bool) bool {
	return !enabled
}

// DisableAlertMessage returns NSAlert copy when disabling a service.
func DisableAlertMessage(running bool) string {
	if running {
		return msgDisableRunning
	}
	return "Server is already stopped"
}

// EnableAlertMessage returns NSAlert copy when enabling a service.
func EnableAlertMessage(running bool) string {
	if running {
		return "Server is already running"
	}
	return msgEnableStopped
}

// FormatServicesEmptyLabel is shown when no services are configured.
func FormatServicesEmptyLabel() string {
	return "No services configured"
}