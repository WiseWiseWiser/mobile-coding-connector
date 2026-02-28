package cloudflare

import (
	"fmt"
	"sync"

	"github.com/xhd2015/lifelog-private/ai-critic/server/logs"
	"github.com/xhd2015/lifelog-private/ai-critic/server/quicktest"
)

const (
	GroupCore      = "core"
	GroupExtension = "extension"
)

type TunnelGroupManager struct {
	mu        sync.RWMutex
	core      *TunnelGroup
	extension *TunnelGroup
}

var (
	groupManager     *TunnelGroupManager
	groupManagerOnce sync.Once
)

func GetTunnelGroupManager() *TunnelGroupManager {
	groupManagerOnce.Do(func() {
		groupManager = &TunnelGroupManager{}
	})
	return groupManager
}

func (m *TunnelGroupManager) GetCoreGroup() *TunnelGroup {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.core == nil {
		tunnelMgr := NewUnifiedTunnelManager(GroupCore)
		m.core = NewTunnelGroup(GroupCore, tunnelMgr)
		fmt.Printf("[tunnel-group-manager] Created core group with tunnel manager\n")
		if quicktest.Enabled() {
			logs.PrintCallerStack()
		}
	}
	return m.core
}

func (m *TunnelGroupManager) GetExtensionGroup() *TunnelGroup {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.extension == nil {
		tunnelMgr := NewUnifiedTunnelManager(GroupExtension)
		m.extension = NewTunnelGroup(GroupExtension, tunnelMgr)
		fmt.Printf("[tunnel-group-manager] Created extension group with tunnel manager\n")
		if quicktest.Enabled() {
			logs.PrintCallerStack()
		}
	}
	return m.extension
}

func (m *TunnelGroupManager) GetGroup(name string) *TunnelGroup {
	switch name {
	case GroupCore:
		return m.GetCoreGroup()
	case GroupExtension:
		return m.GetExtensionGroup()
	default:
		return nil
	}
}
