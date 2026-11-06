package procmon

import (
	"os/user"
	"strings"
)

func protectReason(p rawProc, selfAndAncestors map[int]struct{}) string {
	if p.PID <= 1 {
		return "init (PID <= 1)"
	}
	if _, ok := selfAndAncestors[p.PID]; ok {
		return "this server (self or ancestor)"
	}
	name := displayName(p.Cmd)
	switch name {
	case "WindowServer", "loginwindow", "kernel_task", "launchd":
		return "system-critical (" + name + ")"
	}
	base := strings.ToLower(name)
	if base == "windowserver" || base == "loginwindow" {
		return "system-critical (" + name + ")"
	}
	return ""
}

func tagsFor(p rawProc, username string) []string {
	var tags []string
	if username != "" && p.User == username {
		tags = append(tags, "mine")
	}
	cmd := p.Cmd
	name := displayName(cmd)
	lower := strings.ToLower(cmd)
	if strings.Contains(lower, "cloudflared") {
		tags = append(tags, "tun")
		if strings.Contains(lower, "tunnel --url") || strings.Contains(lower, "trycloudflare") {
			tags = append(tags, "stale")
		}
	}
	if strings.Contains(name, "eluc-wt-bin") || strings.Contains(lower, "eluc-wt-bin") {
		tags = append(tags, "stale")
	}
	return tags
}

func currentUsername() string {
	u, err := user.Current()
	if err != nil {
		return ""
	}
	return u.Username
}

func ancestorSet(selfPID int, byPID map[int]rawProc) map[int]struct{} {
	out := map[int]struct{}{selfPID: {}}
	pid := selfPID
	for i := 0; i < 32; i++ {
		p, ok := byPID[pid]
		if !ok || p.PPID <= 0 {
			break
		}
		out[p.PPID] = struct{}{}
		if p.PPID <= 1 {
			break
		}
		pid = p.PPID
	}
	return out
}
