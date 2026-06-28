//go:build darwin

package wsproxy_singbox

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type proxyEndpoint struct {
	enabled bool
	server  string
	port    int
}

type serviceProxyState struct {
	web    proxyEndpoint
	secure proxyEndpoint
	socks  proxyEndpoint
}

func defaultOutboundBindInterface() string {
	iface, err := defaultRouteInterface()
	if err != nil {
		return ""
	}
	return iface
}

func systemProxyEnabled() bool {
	service, err := activeNetworkService()
	if err != nil {
		return false
	}
	state, err := getServiceProxyState(service)
	if err != nil {
		return false
	}
	return state.web.enabled || state.secure.enabled || state.socks.enabled
}

func disableSystemProxiesForTun() (restore func(), err error) {
	service, err := activeNetworkService()
	if err != nil {
		return nil, err
	}
	previous, err := getServiceProxyState(service)
	if err != nil {
		return nil, err
	}
	if !previous.web.enabled && !previous.secure.enabled && !previous.socks.enabled {
		return func() {}, nil
	}
	if err := setServiceProxyState(service, serviceProxyState{}); err != nil {
		return nil, err
	}
	fmt.Printf("System proxy for %q disabled for TUN (was: %s)\n", service, formatProxyState(previous))
	return func() {
		if restoreErr := setServiceProxyState(service, previous); restoreErr != nil {
			fmt.Fprintf(os.Stderr, "restore system proxy: %v\n", restoreErr)
		} else {
			fmt.Printf("System proxy for %q restored\n", service)
		}
	}, nil
}

func getServiceProxyState(service string) (serviceProxyState, error) {
	web, err := getProxyEndpoint(service, "web")
	if err != nil {
		return serviceProxyState{}, err
	}
	secure, err := getProxyEndpoint(service, "secureweb")
	if err != nil {
		return serviceProxyState{}, err
	}
	socks, err := getProxyEndpoint(service, "socksfirewall")
	if err != nil {
		return serviceProxyState{}, err
	}
	return serviceProxyState{web: web, secure: secure, socks: socks}, nil
}

func getProxyEndpoint(service, kind string) (proxyEndpoint, error) {
	out, err := exec.Command("networksetup", "-get"+kind+"proxy", service).CombinedOutput()
	if err != nil {
		return proxyEndpoint{}, fmt.Errorf("get %s proxy: %w: %s", kind, err, strings.TrimSpace(string(out)))
	}
	return parseProxyEndpoint(string(out)), nil
}

func parseProxyEndpoint(output string) proxyEndpoint {
	var ep proxyEndpoint
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "Enabled:"):
			ep.enabled = strings.TrimSpace(strings.TrimPrefix(line, "Enabled:")) == "Yes"
		case strings.HasPrefix(line, "Server:"):
			ep.server = strings.TrimSpace(strings.TrimPrefix(line, "Server:"))
		case strings.HasPrefix(line, "Port:"):
			if port, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Port:"))); err == nil {
				ep.port = port
			}
		}
	}
	return ep
}

func setServiceProxyState(service string, state serviceProxyState) error {
	if err := setProxyEndpoint(service, "web", state.web); err != nil {
		return err
	}
	if err := setProxyEndpoint(service, "secureweb", state.secure); err != nil {
		return err
	}
	if err := setProxyEndpoint(service, "socksfirewall", state.socks); err != nil {
		return err
	}
	return nil
}

func setProxyEndpoint(service, kind string, ep proxyEndpoint) error {
	if ep.enabled {
		if ep.server == "" || ep.port == 0 {
			return fmt.Errorf("invalid %s proxy endpoint", kind)
		}
		args := []string{"-set" + kind + "proxy", service, ep.server, strconv.Itoa(ep.port)}
		out, err := exec.Command("networksetup", args...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("set %s proxy: %w: %s", kind, err, strings.TrimSpace(string(out)))
		}
		state := "on"
		stateArgs := []string{"-set" + kind + "proxystate", service, state}
		out, err = exec.Command("networksetup", stateArgs...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("enable %s proxy: %w: %s", kind, err, strings.TrimSpace(string(out)))
		}
		return nil
	}
	args := []string{"-set" + kind + "proxystate", service, "off"}
	out, err := exec.Command("networksetup", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("disable %s proxy: %w: %s", kind, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func formatProxyState(state serviceProxyState) string {
	var parts []string
	if state.web.enabled {
		parts = append(parts, fmt.Sprintf("HTTP %s:%d", state.web.server, state.web.port))
	}
	if state.secure.enabled {
		parts = append(parts, fmt.Sprintf("HTTPS %s:%d", state.secure.server, state.secure.port))
	}
	if state.socks.enabled {
		parts = append(parts, fmt.Sprintf("SOCKS %s:%d", state.socks.server, state.socks.port))
	}
	if len(parts) == 0 {
		return "off"
	}
	return strings.Join(parts, ", ")
}