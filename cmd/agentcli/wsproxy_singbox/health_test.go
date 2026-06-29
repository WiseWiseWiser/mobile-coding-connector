package wsproxy_singbox

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestStartWebOutboundHealthMonitorSwitchesOnUpstreamChange(t *testing.T) {
	oldReachable := upstreamReachable
	oldSwitch := switchWebOutbound
	defer func() {
		upstreamReachable = oldReachable
		switchWebOutbound = oldSwitch
	}()

	var reachable atomic.Bool
	reachable.Store(true)
	upstreamReachable = func(port int) bool {
		return reachable.Load()
	}

	var switches []string
	switchWebOutbound = func(outbound string) error {
		switches = append(switches, outbound)
		return nil
	}

	stop := StartWebOutboundHealthMonitor(11080)
	defer stop()

	if len(switches) != 1 || switches[0] != proxyOutboundTag {
		t.Fatalf("initial switches = %v, want [%s]", switches, proxyOutboundTag)
	}

	reachable.Store(false)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		for _, s := range switches {
			if s == directOutboundTag {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("switches = %v, want eventual %s", switches, directOutboundTag)
}