package quicktest

import (
	"fmt"
	"sync"

	"github.com/xhd2015/lifelog-private/ai-critic/server/logs"
)

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

// LogHeavyOperationWithCallerStack prints a contextual message and the caller
// stack trace when quick-test mode is enabled.
// Use this for heavy resource operations that are useful to trace in quick-test:
//   - Starting a long-running server (e.g., opencode web)
//   - Creating or starting tunnel mappings (e.g., cloudflared tunnel)
//   - Starting background tasks or subprocess
func LogHeavyOperationWithCallerStack(format string, args ...any) {
	if !Enabled() {
		return
	}
	fmt.Printf(format, args...)
	if len(format) > 0 && format[len(format)-1] != '\n' {
		fmt.Println()
	}
	logs.PrintCallerStackSkip(1)
}
