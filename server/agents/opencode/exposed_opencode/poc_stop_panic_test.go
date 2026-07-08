//go:build poc

package exposed_opencode

import (
	"sync"
	"testing"
)

// POC tests for script/opencode-stop-panic-poc. Not run in normal CI.
//   go test -tags poc -run TestPOC ./server/agents/opencode/exposed_opencode/

func pocResetManager() {
	managerMutex.Lock()
	manager = nil
	managerMutex.Unlock()
}

func TestPOCStopOnAlreadyClosedChannelIsSafe(t *testing.T) {
	pocResetManager()

	ch := make(chan struct{})
	close(ch)

	managerMutex.Lock()
	manager = &OpencodeManager{Port: 4096, StopChan: ch}
	managerMutex.Unlock()

	Stop()
	Stop()
}

func TestPOCHealthCheckStopWebServerOnClosedChannelIsSafe(t *testing.T) {
	pocResetManager()

	ch := make(chan struct{})
	close(ch)

	managerMutex.Lock()
	manager = &OpencodeManager{Port: 4096, StopChan: ch}
	managerMutex.Unlock()

	if _, err := stopWebServer(&Settings{WebServer: WebServerConfig{Port: 4096}}); err != nil {
		t.Fatalf("stopWebServer: %v", err)
	}
	Stop()
}

func TestPOCConcurrentStopIsSafe(t *testing.T) {
	pocResetManager()

	managerMutex.Lock()
	manager = &OpencodeManager{Port: 4096, StopChan: make(chan struct{})}
	managerMutex.Unlock()

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			Stop()
		}()
	}
	wg.Wait()
}