package qemu

import (
	cfbackend "github.com/xhd2015/dot-pkgs/go-pkgs/cloudflare/backend"
)

// ProductQemuTunnel is the shared guest named-tunnel used when qemu.json is enabled.
// Keeping one tunnel avoids stomping sibling hostnames (agent-fast + owned ports).
const ProductQemuTunnel = "agent-fast-apex-nest-qemu"

// CloudflaredBackend returns host or qemu cloudflared backend from qemu.json.
func CloudflaredBackend() cfbackend.Backend {
	s := cfbackend.Select{Qemu: Enabled()}
	if s.Qemu {
		s.DefaultTunnel = ProductQemuTunnel
		s.QemuManager = DefaultManager()
	}
	return cfbackend.New(s)
}
