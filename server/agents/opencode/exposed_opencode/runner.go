package exposed_opencode

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/tool_exec"
)

var (
	manager         *OpencodeManager
	managerMutex    sync.Mutex
	managerStarting int32
)

type OpencodeManager struct {
	Port     int
	Cmd      *exec.Cmd
	StopChan chan struct{}
}

func isProcessAlive(cmd *exec.Cmd) bool {
	if cmd == nil || cmd.Process == nil {
		return false
	}
	err := syscall.Kill(cmd.Process.Pid, 0)
	return err == nil
}

func isPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

func waitForServer(port int, timeout time.Duration) error {
	url := fmt.Sprintf("http://127.0.0.1:%d/session", port)
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for server on port %d", port)
}

func waitDone(cmd *exec.Cmd) chan struct{} {
	done := make(chan struct{})
	go func() {
		cmd.Wait()
		close(done)
	}()
	return done
}

func StartWithSettings(port int, password string, customPath string) (*OpencodeManager, error) {
	managerMutex.Lock()
	defer managerMutex.Unlock()

	if manager != nil && isProcessAlive(manager.Cmd) {
		return manager, nil
	}

	if manager != nil && manager.StopChan != nil {
		select {
		case <-manager.StopChan:
		default:
			close(manager.StopChan)
		}
		manager = nil
	}

	if atomic.LoadInt32(&managerStarting) == 1 {
		for atomic.LoadInt32(&managerStarting) == 1 {
			time.Sleep(100 * time.Millisecond)
		}
		return manager, nil
	}

	atomic.StoreInt32(&managerStarting, 1)
	defer atomic.StoreInt32(&managerStarting, 0)

	if port == 0 {
		port = 4096
	}

	if !isPortAvailable(port) {
		return nil, fmt.Errorf("port %d is already in use", port)
	}

	manager = &OpencodeManager{
		Port:     port,
		StopChan: make(chan struct{}),
	}

	if err := startServer(manager, password, customPath); err != nil {
		return nil, fmt.Errorf("failed to start opencode server: %w", err)
	}

	if err := waitForServer(port, 10*time.Second); err != nil {
		return nil, fmt.Errorf("opencode server not ready: %w", err)
	}

	fmt.Printf("[exposed_opencode] Server started on port %d\n", port)
	return manager, nil
}

func startServer(mgr *OpencodeManager, password string, customPath string) error {
	cmdOpts := &tool_exec.Options{
		CustomPath: customPath,
	}

	if password != "" {
		cmdOpts.Env = map[string]string{
			"OPENCODE_SERVER_PASSWORD": password,
		}
	}

	cmdWrapper, err := tool_exec.New("opencode", []string{"web", "--port", fmt.Sprintf("%d", mgr.Port)}, cmdOpts)
	if err != nil {
		return fmt.Errorf("failed to create opencode command: %w", err)
	}

	cmd := cmdWrapper.Cmd
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Start()
	if err != nil {
		return err
	}

	mgr.Cmd = cmd

	go func() {
		select {
		case <-mgr.StopChan:
			if mgr.Cmd != nil && mgr.Cmd.Process != nil {
				mgr.Cmd.Process.Kill()
			}
		case <-waitDone(cmd):
		}
	}()

	return nil
}

func Stop() {
	managerMutex.Lock()
	defer managerMutex.Unlock()

	if manager != nil && manager.StopChan != nil {
		fmt.Printf("[exposed_opencode] Stopping server on port %d\n", manager.Port)
		close(manager.StopChan)
		manager.Cmd = nil
	}
}

func GetPort() int {
	managerMutex.Lock()
	defer managerMutex.Unlock()

	if manager != nil && manager.Cmd != nil && manager.Cmd.Process != nil {
		return manager.Port
	}
	return 0
}

func IsRunning() bool {
	managerMutex.Lock()
	defer managerMutex.Unlock()

	if manager != nil && isProcessAlive(manager.Cmd) {
		return true
	}
	return false
}
