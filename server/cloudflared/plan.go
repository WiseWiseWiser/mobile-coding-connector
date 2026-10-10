// Package cloudflared switches an origin among native cloudflared, qemu
// cloudflared, and the edge cloudflare-proxy. Plan is pure; Apply performs it.
package cloudflared

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Backend is the one connector that should own public hostnames.
type Backend string

const (
	BackendNative Backend = "native"
	BackendQemu   Backend = "qemu"
	BackendProxy  Backend = "proxy"
)

// Step is one reconcile action, in the order Apply runs them.
type Step string

const (
	StepWrite     Step = "write"
	StepRepublish Step = "republish"
	StepVerify    Step = "verify"
	StepStopGuest Step = "stop-guest"
	StepStopProxy Step = "stop-proxy"
	StepStopHost  Step = "stop-host"
)

// Files is the on-disk selector. Mode "" means native. Token is presence only.
type Files struct {
	Mode        string
	ProxyURL    string
	HasToken    bool
	QemuEnabled bool
}

// Runtime is what is actually running. It is not the DNS target; a live guest
// cloudflared is treated as the connector that last held the public hostname.
type Runtime struct {
	GuestCloudflared         bool
	GuestCFPID               string
	ProxyDials               int
	ProxySessions            int
	HostCloudflared          bool
	HostCloudflaredInstalled bool
	QemuInstalled            bool
	PublicHost               string
}

// Switch is the ordered plan for one use. Noop means the target is already
// the configured backend and the old connector is already gone.
type Switch struct {
	Target Backend
	Noop   bool
	Steps  []Step
}

// ParseBackend accepts native, qemu, or proxy.
func ParseBackend(s string) (Backend, error) {
	switch Backend(strings.ToLower(strings.TrimSpace(s))) {
	case BackendNative:
		return BackendNative, nil
	case BackendQemu:
		return BackendQemu, nil
	case BackendProxy:
		return BackendProxy, nil
	default:
		return "", fmt.Errorf("backend must be native, qemu, or proxy")
	}
}

// ConfiguredBackend is the backend the files select. Proxy wins over qemu
// when both are set, matching StartDomainTunnel.
func ConfiguredBackend(f Files) Backend {
	if strings.EqualFold(strings.TrimSpace(f.Mode), "proxy") {
		return BackendProxy
	}
	if f.QemuEnabled {
		return BackendQemu
	}
	return BackendNative
}

// EffectiveBackend is the connector that is still in a position to serve.
// Guest cloudflared wins while its process is up, because its tunnel route
// is what public DNS follows until that process is stopped.
func EffectiveBackend(f Files, rt Runtime) Backend {
	if rt.GuestCloudflared {
		return BackendQemu
	}
	if rt.ProxySessions > 0 || (ConfiguredBackend(f) == BackendProxy && rt.ProxyDials > 0) {
		return BackendProxy
	}
	if rt.HostCloudflared {
		return BackendNative
	}
	return ConfiguredBackend(f)
}

// PlanSwitch returns the reconcile steps. A missing proxy token or a missing
// binary is an error and Steps is empty, so the caller must not write files.
func PlanSwitch(f Files, rt Runtime, target Backend) (Switch, error) {
	switch target {
	case BackendProxy:
		if strings.TrimSpace(f.ProxyURL) == "" || !f.HasToken {
			return Switch{}, fmt.Errorf("cloudflare mode=proxy requires proxy_url and token")
		}
	case BackendQemu:
		if !rt.QemuInstalled {
			return Switch{}, fmt.Errorf("qemu is not installed")
		}
	case BackendNative:
		if !rt.HostCloudflaredInstalled {
			return Switch{}, fmt.Errorf("cloudflared is not installed")
		}
	default:
		return Switch{}, fmt.Errorf("backend must be native, qemu, or proxy")
	}

	sw := Switch{Target: target}
	if alreadyOn(f, rt, target) {
		sw.Noop = true
		return sw, nil
	}

	sw.Steps = append(sw.Steps, StepWrite, StepRepublish, StepVerify)
	switch target {
	case BackendProxy:
		if rt.GuestCloudflared {
			sw.Steps = append(sw.Steps, StepStopGuest)
		}
		if rt.HostCloudflared {
			sw.Steps = append(sw.Steps, StepStopHost)
		}
	case BackendQemu:
		if rt.ProxySessions > 0 || rt.ProxyDials > 0 {
			sw.Steps = append(sw.Steps, StepStopProxy)
		}
		if rt.HostCloudflared {
			sw.Steps = append(sw.Steps, StepStopHost)
		}
	case BackendNative:
		if rt.GuestCloudflared {
			sw.Steps = append(sw.Steps, StepStopGuest)
		}
		if rt.ProxySessions > 0 || rt.ProxyDials > 0 {
			sw.Steps = append(sw.Steps, StepStopProxy)
		}
	}
	return sw, nil
}

func alreadyOn(f Files, rt Runtime, target Backend) bool {
	if ConfiguredBackend(f) != target {
		return false
	}
	switch target {
	case BackendProxy:
		return !f.QemuEnabled && !rt.GuestCloudflared && !rt.HostCloudflared && (rt.ProxyDials > 0 || rt.ProxySessions > 0)
	case BackendQemu:
		return rt.GuestCloudflared && rt.ProxySessions == 0 && rt.ProxyDials == 0 && !rt.HostCloudflared
	case BackendNative:
		return rt.HostCloudflared && !rt.GuestCloudflared && rt.ProxySessions == 0 && rt.ProxyDials == 0
	default:
		return false
	}
}

// StatusLine is one stdout row. Warning is empty when files and the running
// connector agree.
type StatusLine struct {
	Lines   []string
	Warning string
}

// FormatStatus renders `cloudflared status`.
func FormatStatus(f Files, rt Runtime) StatusLine {
	configured := ConfiguredBackend(f)
	effective := EffectiveBackend(f, rt)
	lines := []string{
		row("backend:", string(effective)),
		row("configured:", string(configured)),
		row("qemu:", qemuStatus(f, rt)),
		row("proxy:", proxyStatus(f)),
		row("public:", publicURL(rt.PublicHost)),
	}
	out := StatusLine{Lines: lines}
	if configured != effective {
		out.Warning = fmt.Sprintf("configured backend %s but public DNS still hits the other connector", configured)
	}
	return out
}

// UseSummary is the success block printed after a switch.
type UseSummary struct {
	Target    Backend
	ProxyURL  string
	QemuLine  string
	Published string
	Ping      int
}

// FormatUse renders the success block for `cloudflared use`.
func FormatUse(s UseSummary) []string {
	lines := []string{row("backend:", string(s.Target))}
	if s.Target == BackendProxy {
		lines = append(lines, row("proxy_url:", strings.TrimSpace(s.ProxyURL)))
	}
	if s.QemuLine != "" {
		lines = append(lines, row("qemu:", s.QemuLine))
	}
	if s.Target != BackendProxy {
		lines = append(lines, row("proxy:", "off"))
	}
	lines = append(lines,
		row("published:", s.Published),
		row("ping:", fmt.Sprintf("%d", s.Ping)),
	)
	return lines
}

func qemuStatus(f Files, rt Runtime) string {
	if !f.QemuEnabled {
		if rt.GuestCloudflared {
			return "disabled  cloudflared still running"
		}
		return "disabled"
	}
	if rt.GuestCloudflared && rt.GuestCFPID != "" {
		return "enabled  cloudflared pid " + rt.GuestCFPID
	}
	if rt.GuestCloudflared {
		return "enabled  cloudflared running"
	}
	return "enabled  cloudflared down"
}

func proxyStatus(f Files) string {
	if ConfiguredBackend(f) != BackendProxy {
		return "off"
	}
	if u := strings.TrimSpace(f.ProxyURL); u != "" {
		return u
	}
	return "on"
}

func publicURL(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return "(none)"
	}
	return "https://" + host
}

func row(label, value string) string {
	return fmt.Sprintf("%-12s%s", label, value)
}

// GuestProbeTimeout bounds a guest SSH status probe. Past this the command
// keeps the lines it already printed and warns instead of waiting.
const GuestProbeTimeout = 15 * time.Second

// GuestCF is the guest cloudflared probe result. Alive is false when the
// probe timed out or the process is down.
type GuestCF struct {
	Alive bool
	PID   string
	Err   error
}

// Dials is the edge dial-pool total. Err is non-fatal: status still prints.
type Dials struct {
	Count int
	Err   error
}

// StatusSink receives one finished fact. Warn is a stderr warning without the
// "warning: " prefix.
type StatusSink struct {
	Emit func(line string)
	Warn func(message string)
}

// GuestProbe and DialProbe are the slow checks. They run only after the file
// facts have been emitted.
type GuestProbe func(ctx context.Context) GuestCF
type DialProbe func(ctx context.Context) Dials

// RunCloudflaredStatus emits file facts before calling the probes, then the
// guest and edge lines, and backend last.
func RunCloudflaredStatus(ctx context.Context, f Files, public string, guest GuestProbe, dials DialProbe, sink StatusSink) {
	emit, warn := sink.funcs()
	emitFileStatus(f, public, emit)
	gctx, cancel := context.WithTimeout(ctx, GuestProbeTimeout)
	g := GuestCF{}
	if guest != nil {
		g = guest(gctx)
	}
	cancel()
	d := Dials{}
	if dials != nil {
		d = dials(ctx)
	}
	emitCloudflaredTail(f, public, g, d, emit, warn)
}

func (s StatusSink) funcs() (emit func(string), warn func(string)) {
	emit = s.Emit
	if emit == nil {
		emit = func(string) {}
	}
	warn = s.Warn
	if warn == nil {
		warn = func(string) {}
	}
	return emit, warn
}

func emitFileStatus(f Files, public string, emit func(string)) {
	emit(row("configured:", string(ConfiguredBackend(f))))
	if f.QemuEnabled {
		emit(row("qemu:", "enabled"))
	} else {
		emit(row("qemu:", "disabled"))
	}
	emit(row("proxy:", proxyStatus(f)))
	emit(row("public:", publicURL(public)))
}

func emitCloudflaredTail(f Files, public string, guest GuestCF, dials Dials, emit func(string), warn func(string)) {
	rt := Runtime{PublicHost: public}
	if errors.Is(guest.Err, context.DeadlineExceeded) {
		warn("guest ssh timed out after " + GuestProbeTimeout.String())
		emit(row("qemu:", "guest cloudflared unknown"))
	} else if guest.Err != nil {
		warn("guest cloudflared: " + guest.Err.Error())
		emit(row("qemu:", "guest cloudflared unknown"))
	} else if guest.Alive {
		rt.GuestCloudflared = true
		rt.GuestCFPID = guest.PID
		if guest.PID != "" {
			emit(row("qemu:", "guest cloudflared pid "+guest.PID))
		} else {
			emit(row("qemu:", "guest cloudflared running"))
		}
	} else {
		emit(row("qemu:", "guest cloudflared down"))
	}

	if dials.Err != nil {
		warn("edge dials: " + dials.Err.Error())
	} else {
		rt.ProxyDials = dials.Count
	}
	emit(row("backend:", string(EffectiveBackend(f, rt))))
}
