package exposed_opencode

import (
	"sync"
	"testing"
)

func TestStopIsIdempotent(t *testing.T) {
	managerMutex.Lock()
	manager = &OpencodeManager{
		Port:     4096,
		StopChan: make(chan struct{}),
	}
	managerMutex.Unlock()

	Stop()
	Stop()
}

func TestStopOnAlreadyClosedChannel(t *testing.T) {
	ch := make(chan struct{})
	close(ch)

	managerMutex.Lock()
	manager = &OpencodeManager{Port: 4096, StopChan: ch}
	managerMutex.Unlock()

	Stop()
	Stop()
}

func TestConcurrentStop(t *testing.T) {
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