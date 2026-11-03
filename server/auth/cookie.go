package auth

import (
	"sync"

	"github.com/xhd2015/ai-critic/server/auth/cookiename"
)

const (
	// DefaultCookieName preserves the browser authentication protocol of existing instances.
	DefaultCookieName = cookiename.DefaultName
	// ServiceCookieName is reserved for the separate service/mapping authentication gate.
	ServiceCookieName = cookiename.ServiceName
)

var managementCookie = struct {
	sync.RWMutex
	name string
}{name: DefaultCookieName}

// CookieName returns this process's management authentication cookie name.
func CookieName() string {
	managementCookie.RLock()
	defer managementCookie.RUnlock()
	return managementCookie.name
}

// ValidateCookieName validates a management cookie name without changing configuration.
// Secure browser prefixes are not supported because this server does not issue
// Secure cookies (TLS can terminate at an external public proxy).
func ValidateCookieName(name string) error {
	return cookiename.Validate(name)
}

// SetCookieName configures the process before registering/serving HTTP handlers.
// It is a startup setting, not a runtime or per-mapping override. Invalid input
// leaves the previous setting intact. Other instances' cookie names are not aliases.
func SetCookieName(name string) error {
	if err := ValidateCookieName(name); err != nil {
		return err
	}
	managementCookie.Lock()
	defer managementCookie.Unlock()
	managementCookie.name = name
	return nil
}
