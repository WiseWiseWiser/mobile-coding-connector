package cloudflare

import (
	"crypto/md5"
	"fmt"
	"net"
	"os"
	"strings"
)

const (
	MaxTunnelNameLength = 32
	TunnelNamePrefix    = "mcc-"
)

func GenerateTunnelName(group string) string {
	hostname := getHostname()
	hostIP := getHostIP()

	uuid := generateUUID(hostname, hostIP)

	prefix := TunnelNamePrefix
	suffix := "-" + group

	remaining := MaxTunnelNameLength - len(prefix) - len(suffix) - 1 - len(hostIP) - 1

	if remaining < 4 {
		hostname = ""
		hostIP = strings.ReplaceAll(hostIP, ".", "-")
		uuid = uuid[:8]
		return fmt.Sprintf("%s%s-%s%s", prefix, uuid, hostIP, suffix)
	}

	if len(hostname) > remaining/2 {
		hostname = hostname[:remaining/2]
	}

	uuidLen := remaining - len(hostname) - 1
	if uuidLen < 4 {
		uuidLen = 4
	}
	if uuidLen > len(uuid) {
		uuidLen = len(uuid)
	}

	hostname = strings.Map(sanitizeRune, hostname)
	hostIP = strings.Map(sanitizeRune, hostIP)

	return fmt.Sprintf("%s%s-%s-%s%s", prefix, hostname, hostIP, uuid[:uuidLen], suffix)
}

func sanitizeRune(r rune) rune {
	if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
		return r
	}
	return '-'
}

func getHostname() string {
	if h, err := os.Hostname(); err == nil {
		return h
	}
	return "host"
}

func getHostIP() string {
	if addrs, err := net.InterfaceAddrs(); err == nil {
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					return ipnet.IP.String()
				}
			}
		}
	}
	return "0.0.0.0"
}

func generateUUID(hostname, hostIP string) string {
	data := hostname + hostIP
	hash := md5.Sum([]byte(data))
	return fmt.Sprintf("%x", hash)
}
