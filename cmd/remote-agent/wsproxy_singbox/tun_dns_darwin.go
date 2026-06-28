//go:build darwin

package wsproxy_singbox

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var networkServiceOrderLine = regexp.MustCompile(`^\(\d+\)\s+(.+)$`)

func restoreStuckTunDNS() error {
	service, err := activeNetworkService()
	if err != nil {
		return nil
	}
	servers, err := getDNSServers(service)
	if err != nil {
		return nil
	}
	for _, server := range servers {
		if server == tunDNSAddress {
			if err := restoreDNSServers(service, nil); err != nil {
				return fmt.Errorf("restore stale TUN DNS: %w", err)
			}
			fmt.Printf("Restored system DNS for %q from stale TUN address %s\n", service, tunDNSAddress)
			return nil
		}
	}
	return nil
}

func configurePlatformTunDNS() (restore func(), err error) {
	service, err := activeNetworkService()
	if err != nil {
		return nil, err
	}
	previous, err := getDNSServers(service)
	if err != nil {
		return nil, err
	}
	if err := setDNSServers(service, []string{tunDNSAddress}); err != nil {
		return nil, err
	}
	fmt.Printf("System DNS for %q set to %s (was: %s)\n", service, tunDNSAddress, formatDNSServers(previous))
	return func() {
		if restoreErr := restoreDNSServers(service, previous); restoreErr != nil {
			fmt.Fprintf(os.Stderr, "restore system DNS: %v\n", restoreErr)
		} else {
			fmt.Printf("System DNS for %q restored\n", service)
		}
	}, nil
}

func activeNetworkService() (string, error) {
	iface, err := defaultRouteInterface()
	if err != nil {
		return "", err
	}
	return networkServiceForDevice(iface)
}

func defaultRouteInterface() (string, error) {
	out, err := exec.Command("route", "-n", "get", "default").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("default route: %w: %s", err, strings.TrimSpace(string(out)))
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "interface:") {
			iface := strings.TrimSpace(strings.TrimPrefix(line, "interface:"))
			if iface != "" {
				return iface, nil
			}
		}
	}
	return "", fmt.Errorf("default route interface not found")
}

func networkServiceForDevice(device string) (string, error) {
	out, err := exec.Command("networksetup", "-listnetworkserviceorder").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("list network services: %w", err)
	}
	if service, ok := parseNetworkServiceForDevice(string(out), device); ok {
		return service, nil
	}
	return "", fmt.Errorf("network service for %s not found", device)
}

func parseNetworkServiceForDevice(listOutput, device string) (string, bool) {
	want := "Device: " + device
	var current string
	for _, line := range strings.Split(listOutput, "\n") {
		trimmed := strings.TrimSpace(line)
		if m := networkServiceOrderLine.FindStringSubmatch(trimmed); m != nil {
			current = strings.TrimSpace(m[1])
			continue
		}
		if strings.Contains(trimmed, want) && current != "" {
			return current, true
		}
	}
	return "", false
}

func getDNSServers(service string) ([]string, error) {
	out, err := exec.Command("networksetup", "-getdnsservers", service).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("get DNS servers: %w: %s", err, strings.TrimSpace(string(out)))
	}
	text := strings.TrimSpace(string(out))
	if text == "" || strings.Contains(text, "There aren't any DNS Servers set") {
		return nil, nil
	}
	return strings.Split(text, "\n"), nil
}

func setDNSServers(service string, servers []string) error {
	args := append([]string{"-setdnsservers", service}, servers...)
	out, err := exec.Command("networksetup", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("set DNS servers: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func restoreDNSServers(service string, previous []string) error {
	if len(previous) == 0 {
		out, err := exec.Command("networksetup", "-setdnsservers", service, "Empty").CombinedOutput()
		if err != nil {
			return fmt.Errorf("restore DHCP DNS: %w: %s", err, strings.TrimSpace(string(out)))
		}
		return nil
	}
	return setDNSServers(service, previous)
}

func formatDNSServers(servers []string) string {
	if len(servers) == 0 {
		return "DHCP"
	}
	return strings.Join(servers, ", ")
}