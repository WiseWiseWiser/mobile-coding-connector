package quicktest

import "sync"

var (
	mu          sync.RWMutex
	enabled     bool
	keepEnabled bool
)

func SetEnabled(v bool) {
	mu.Lock()
	defer mu.Unlock()
	enabled = v
}

func Enabled() bool {
	mu.RLock()
	defer mu.RUnlock()
	return enabled
}

func SetKeepEnabled(v bool) {
	mu.Lock()
	defer mu.Unlock()
	keepEnabled = v
}

func KeepEnabled() bool {
	mu.RLock()
	defer mu.RUnlock()
	return keepEnabled
}
