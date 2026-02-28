package internal_opencode

import (
	"fmt"
	"net"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	common "github.com/xhd2015/lifelog-private/ai-critic/server/agents/opencode/common_opencode"
	"github.com/xhd2015/lifelog-private/ai-critic/server/logs"
	"github.com/xhd2015/lifelog-private/ai-critic/server/quicktest"
)

var (
	serverInstance *OpencodeServer
	serverMutex    sync.Mutex
	starting       int32 // atomic: 0 = not starting, 1 = starting
)

// OpencodeServer holds the state of a running internal opencode server.
type OpencodeServer struct {
	Port     int
	Cmd      *exec.Cmd
	StopChan chan struct{}
}

// GetOrStartOpencodeServer returns the existing internal opencode server or starts a new one.
func GetOrStartOpencodeServer() (*OpencodeServer, error) {
	serverMutex.Lock()
	defer serverMutex.Unlock()

	if serverInstance != nil {
		if common.IsCmdAlive(serverInstance.Cmd) {
			return serverInstance, nil
		}
		// Reused registry-backed instances may not carry Cmd in-memory.
		if serverInstance.Cmd == nil && serverInstance.Port > 0 && IsPortReachable(serverInstance.Port) {
			return serverInstance, nil
		}
	}

	if serverInstance != nil && serverInstance.StopChan != nil {
		select {
		case <-serverInstance.StopChan:
		default:
			close(serverInstance.StopChan)
		}
		serverInstance = nil
	}

	if atomic.LoadInt32(&starting) == 1 {
		for atomic.LoadInt32(&starting) == 1 {
			time.Sleep(100 * time.Millisecond)
		}
		return serverInstance, nil
	}

	atomic.StoreInt32(&starting, 1)
	defer atomic.StoreInt32(&starting, 0)

	var result *OpencodeServer
	resultErr := WithLock(func() error {
		info, err := LoadRegistry()
		if err != nil {
			fmt.Printf("[opencode] Warning: failed to load registry: %v\n", err)
		}

		if info != nil && info.PID > 0 && info.Port > 0 {
			if IsProcessAlive(info.PID) && IsPortReachable(info.Port) {
				fmt.Printf("[opencode] Reusing existing internal server: PID=%d, Port=%d\n", info.PID, info.Port)
				if quicktest.Enabled() {
					fmt.Printf("[opencode] Reusing existing internal server caller:\n")
					logs.PrintCallerStack()
				}
				result = &OpencodeServer{
					Port: info.Port,
				}
				return nil
			}
			fmt.Printf("[opencode] Registry process dead or port unreachable: PID=%d, Port=%d\n", info.PID, info.Port)
		}

		port, err := findAvailablePort()
		if err != nil {
			return fmt.Errorf("failed to find available port: %w", err)
		}

		newServer := &OpencodeServer{
			Port:     port,
			StopChan: make(chan struct{}),
		}

		if err := startOpencodeWebServer(newServer); err != nil {
			return fmt.Errorf("failed to start opencode server: %w", err)
		}

		if err := common.WaitForSessionReady(port, 10*time.Second); err != nil {
			return fmt.Errorf("opencode server not ready: %w", err)
		}

		if newServer.Cmd != nil && newServer.Cmd.Process != nil {
			regInfo := &InternalServerInfo{
				PID:       newServer.Cmd.Process.Pid,
				Port:      port,
				StartTime: time.Now().Unix(),
			}
			if err := SaveRegistry(regInfo); err != nil {
				fmt.Printf("[opencode] Warning: failed to save registry: %v\n", err)
			}
		}

		fmt.Printf("[opencode] Internal server started on port %d\n", port)
		result = newServer
		return nil
	})
	if resultErr != nil {
		return nil, resultErr
	}

	serverInstance = result
	return result, nil
}

func findAvailablePort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	return port, nil
}

func startOpencodeWebServer(server *OpencodeServer) error {
	cmd, err := common.StartWebProcess(server.Port, nil, server.StopChan)
	if err != nil {
		return err
	}

	server.Cmd = cmd
	return nil
}

// ShutdownOpencodeServer stops the internal opencode server.
func ShutdownOpencodeServer() {
	serverMutex.Lock()
	defer serverMutex.Unlock()

	if serverInstance != nil && serverInstance.StopChan != nil {
		fmt.Printf("[opencode] Stopping server on port %d\n", serverInstance.Port)
		close(serverInstance.StopChan)
		serverInstance.Cmd = nil
	}
}

// GetRunningServerPort returns the port of the currently running server, or 0 if not running.
func GetRunningServerPort() int {
	serverMutex.Lock()
	defer serverMutex.Unlock()

	if serverInstance != nil {
		if common.IsCmdAlive(serverInstance.Cmd) {
			return serverInstance.Port
		}
		if serverInstance.Cmd == nil && serverInstance.Port > 0 && IsPortReachable(serverInstance.Port) {
			return serverInstance.Port
		}
	}

	info, err := LoadRegistry()
	if err == nil && info != nil && info.PID > 0 && info.Port > 0 && IsProcessAlive(info.PID) && IsPortReachable(info.Port) {
		return info.Port
	}
	return 0
}
