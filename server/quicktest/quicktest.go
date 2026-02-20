package quicktest

import "sync"

var (
	mu                sync.RWMutex
	enabled           bool
	keepEnabled       bool
	execRestartBinary string
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

func SetExecRestartBinary(path string) {
	mu.Lock()
	defer mu.Unlock()
	execRestartBinary = path
}

func GetExecRestartBinary() string {
	mu.RLock()
	defer mu.RUnlock()
	return execRestartBinary
}
