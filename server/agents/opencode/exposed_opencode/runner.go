package exposed_opencode

import (
	"fmt"
	"net"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	common "github.com/xhd2015/lifelog-private/ai-critic/server/agents/opencode/common_opencode"
	"github.com/xhd2015/lifelog-private/ai-critic/server/proc_manager"
	"github.com/xhd2015/lifelog-private/ai-critic/server/tool_exec"
)

const procName = "opencode-web"

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

func IsPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

func StartWithSettings(port int, password string, customPath string) (*OpencodeManager, error) {
	managerMutex.Lock()
	defer managerMutex.Unlock()

	var startErr error
	err := proc_manager.WithLock(procName, func() error {
		reg, err := proc_manager.LoadRegistry(procName)
		if err != nil {
			fmt.Printf("[exposed_opencode] Warning: failed to load registry: %v\n", err)
		}

		if reg != nil && reg.PID > 0 && proc_manager.IsProcessAlive(reg.PID) {
			isReachable := proc_manager.IsPortReachable(reg.Port, "/session")
			if isReachable && reg.Port == port {
				fmt.Printf("[exposed_opencode] Reusing existing server: PID=%d, Port=%d\n", reg.PID, reg.Port)
				manager = &OpencodeManager{
					Port: reg.Port,
					Cmd:  nil,
				}
				return nil
			}
			fmt.Printf("[exposed_opencode] Existing server dead or port mismatch: PID=%d, Port=%d, expected=%d\n", reg.PID, reg.Port, port)
			proc_manager.StopProcess(reg.PID)
		}

		if atomic.LoadInt32(&managerStarting) == 1 {
			for atomic.LoadInt32(&managerStarting) == 1 {
				time.Sleep(100 * time.Millisecond)
			}
			return nil
		}

		atomic.StoreInt32(&managerStarting, 1)
		defer atomic.StoreInt32(&managerStarting, 0)

		if port == 0 {
			port = 4096
		}

		if !IsPortAvailable(port) {
			return fmt.Errorf("port %d is already in use", port)
		}

		manager = &OpencodeManager{
			Port:     port,
			StopChan: make(chan struct{}),
		}

		if err := startServer(manager, password, customPath); err != nil {
			startErr = fmt.Errorf("failed to start opencode server: %w", err)
			return err
		}

		if err := common.WaitForSessionReady(port, 10*time.Second); err != nil {
			startErr = fmt.Errorf("opencode server not ready: %w", err)
			return err
		}

		err = proc_manager.SaveRegistry(procName, &proc_manager.ProcessRegistry{
			Name:       procName,
			PID:        manager.Cmd.Process.Pid,
			Port:       port,
			StartTime:  time.Now().Unix(),
			CustomPath: customPath,
		})
		if err != nil {
			fmt.Printf("[exposed_opencode] Warning: failed to save registry: %v\n", err)
		}

		fmt.Printf("[exposed_opencode] Server started on port %d\n", port)
		return nil
	})

	if err != nil {
		return nil, err
	}
	if startErr != nil {
		return nil, startErr
	}
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

	cmd, err := common.StartWebProcess(mgr.Port, cmdOpts, mgr.StopChan)
	if err != nil {
		return err
	}

	mgr.Cmd = cmd
	return nil
}

func Stop() {
	managerMutex.Lock()
	defer managerMutex.Unlock()

	proc_manager.WithLock(procName, func() error {
		reg, _ := proc_manager.LoadRegistry(procName)
		if reg != nil && reg.PID > 0 {
			proc_manager.StopProcess(reg.PID)
		}
		proc_manager.ClearRegistry(procName)
		return nil
	})

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

	reg, err := proc_manager.LoadRegistry(procName)
	if err == nil && reg != nil && proc_manager.IsProcessAlive(reg.PID) {
		return reg.Port
	}
	return 0
}

func IsRunning() bool {
	managerMutex.Lock()
	defer managerMutex.Unlock()

	if manager != nil && common.IsCmdAlive(manager.Cmd) {
		return true
	}

	reg, err := proc_manager.LoadRegistry(procName)
	if err == nil && reg != nil && proc_manager.IsProcessAlive(reg.PID) {
		return true
	}
	return false
}
