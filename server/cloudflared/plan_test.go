package cloudflared

import (
	"context"
	"strings"
	"testing"
	"time"
)

func proxyFiles() Files {
	return Files{Mode: "proxy", ProxyURL: "https://cfproxy-aes423.xhd2015.xyz", HasToken: true}
}

func TestPlanProxyMissingToken(t *testing.T) {
	_, err := PlanSwitch(Files{ProxyURL: "https://edge.example"}, Runtime{}, BackendProxy)
	if err == nil || err.Error() != "cloudflare mode=proxy requires proxy_url and token" {
		t.Fatalf("err = %v", err)
	}
	_, err = PlanSwitch(Files{HasToken: true}, Runtime{}, BackendProxy)
	if err == nil || !strings.Contains(err.Error(), "proxy_url and token") {
		t.Fatalf("err = %v", err)
	}
}

func TestPlanProxyFromQemu(t *testing.T) {
	f := Files{QemuEnabled: true, ProxyURL: "https://cfproxy-aes423.xhd2015.xyz", HasToken: true}
	rt := Runtime{GuestCloudflared: true, GuestCFPID: "2082", QemuInstalled: true, HostCloudflaredInstalled: true}
	sw, err := PlanSwitch(f, rt, BackendProxy)
	if err != nil {
		t.Fatal(err)
	}
	if sw.Noop {
		t.Fatal("expected work")
	}
	got := strings.Join(stepNames(sw.Steps), ",")
	want := "write,republish,verify,stop-guest"
	if got != want {
		t.Fatalf("steps = %s, want %s", got, want)
	}
}

func TestPlanProxyNoop(t *testing.T) {
	rt := Runtime{ProxyDials: 32, ProxySessions: 1, QemuInstalled: true, HostCloudflaredInstalled: true}
	sw, err := PlanSwitch(proxyFiles(), rt, BackendProxy)
	if err != nil {
		t.Fatal(err)
	}
	if !sw.Noop || len(sw.Steps) != 0 {
		t.Fatalf("noop = %v steps = %v", sw.Noop, sw.Steps)
	}
}

func TestPlanProxyRepublishWhenDialsDead(t *testing.T) {
	sw, err := PlanSwitch(proxyFiles(), Runtime{QemuInstalled: true, HostCloudflaredInstalled: true}, BackendProxy)
	if err != nil {
		t.Fatal(err)
	}
	if sw.Noop {
		t.Fatal("dead dials must republish")
	}
	if strings.Join(stepNames(sw.Steps), ",") != "write,republish,verify" {
		t.Fatalf("steps = %v", sw.Steps)
	}
}

func TestPlanQemuStopsProxy(t *testing.T) {
	f := proxyFiles()
	rt := Runtime{
		ProxyDials: 32, ProxySessions: 1,
		QemuInstalled: true, HostCloudflaredInstalled: true,
	}
	sw, err := PlanSwitch(f, rt, BackendQemu)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(stepNames(sw.Steps), ",")
	if got != "write,republish,verify,stop-proxy" {
		t.Fatalf("steps = %s", got)
	}
}

func TestPlanQemuMissingBinary(t *testing.T) {
	_, err := PlanSwitch(Files{}, Runtime{}, BackendQemu)
	if err == nil || err.Error() != "qemu is not installed" {
		t.Fatalf("err = %v", err)
	}
}

func TestPlanNative(t *testing.T) {
	f := Files{QemuEnabled: true}
	rt := Runtime{GuestCloudflared: true, ProxyDials: 4, QemuInstalled: true, HostCloudflaredInstalled: true}
	sw, err := PlanSwitch(f, rt, BackendNative)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(stepNames(sw.Steps), ",")
	if got != "write,republish,verify,stop-guest,stop-proxy" {
		t.Fatalf("steps = %s", got)
	}
}

func TestPlanNativeMissingBinary(t *testing.T) {
	_, err := PlanSwitch(Files{}, Runtime{QemuInstalled: true}, BackendNative)
	if err == nil || err.Error() != "cloudflared is not installed" {
		t.Fatalf("err = %v", err)
	}
}

func TestFormatStatusAgrees(t *testing.T) {
	f := Files{QemuEnabled: true}
	rt := Runtime{GuestCloudflared: true, GuestCFPID: "2082", PublicHost: "agent-fast-apex-nest-23aed.xhd2015.xyz"}
	got := FormatStatus(f, rt)
	text := strings.Join(got.Lines, "\n")
	for _, want := range []string{
		"backend:    qemu",
		"configured: qemu",
		"qemu:       enabled  cloudflared pid 2082",
		"proxy:      off",
		"public:     https://agent-fast-apex-nest-23aed.xhd2015.xyz",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in\n%s", want, text)
		}
	}
	if got.Warning != "" {
		t.Fatalf("warning = %q", got.Warning)
	}
}

func TestFormatStatusDisagrees(t *testing.T) {
	f := Files{QemuEnabled: true}
	rt := Runtime{ProxyDials: 32, ProxySessions: 1, PublicHost: "app.example"}
	got := FormatStatus(f, rt)
	if got.Warning != "configured backend qemu but public DNS still hits the other connector" {
		t.Fatalf("warning = %q", got.Warning)
	}
	if !strings.Contains(strings.Join(got.Lines, "\n"), "backend:    proxy") {
		t.Fatalf("lines:\n%s", strings.Join(got.Lines, "\n"))
	}
}

func TestFormatUseProxy(t *testing.T) {
	lines := FormatUse(UseSummary{
		Target:    BackendProxy,
		ProxyURL:  "https://cfproxy-aes423.xhd2015.xyz",
		QemuLine:  "disabled  guest cloudflared stopped",
		Published: "agent-fast-apex-nest-23aed.xhd2015.xyz",
		Ping:      200,
	})
	text := strings.Join(lines, "\n")
	for _, want := range []string{
		"backend:    proxy",
		"proxy_url:  https://cfproxy-aes423.xhd2015.xyz",
		"qemu:       disabled  guest cloudflared stopped",
		"published:  agent-fast-apex-nest-23aed.xhd2015.xyz",
		"ping:       200",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in\n%s", want, text)
		}
	}
}

func TestRunCloudflaredStatusPrintsFilesBeforeGuest(t *testing.T) {
	var lines []string
	guestEntered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		RunCloudflaredStatus(context.Background(), proxyFiles(), "app.example",
			func(ctx context.Context) GuestCF {
				close(guestEntered)
				select {
				case <-release:
					return GuestCF{}
				case <-ctx.Done():
					return GuestCF{Err: ctx.Err()}
				}
			},
			func(context.Context) Dials { return Dials{Count: 4} },
			StatusSink{Emit: func(line string) { lines = append(lines, line) }},
		)
	}()
	select {
	case <-guestEntered:
	case <-time.After(time.Second):
		t.Fatal("guest probe was not started")
	}
	if len(lines) < 4 {
		t.Fatalf("file lines not emitted before guest returned: %v", lines)
	}
	for _, line := range lines {
		if strings.HasPrefix(line, "backend:") || strings.Contains(line, "guest cloudflared") {
			t.Fatalf("slow line emitted before guest returned: %v", lines)
		}
	}
	close(release)
	<-done
	joined := strings.Join(lines, "\n")
	for _, want := range []string{
		"configured: proxy",
		"qemu:       disabled",
		"proxy:      https://cfproxy-aes423.xhd2015.xyz",
		"public:     https://app.example",
		"qemu:       guest cloudflared down",
		"backend:    proxy",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in\n%s", want, joined)
		}
	}
	if !strings.HasPrefix(lines[len(lines)-1], "backend:") {
		t.Fatalf("last line = %q", lines[len(lines)-1])
	}
}

func TestRunCloudflaredStatusGuestTimeoutWarns(t *testing.T) {
	var lines []string
	var warnings []string
	RunCloudflaredStatus(context.Background(), proxyFiles(), "app.example",
		func(context.Context) GuestCF {
			return GuestCF{Err: context.DeadlineExceeded}
		},
		func(context.Context) Dials { return Dials{} },
		StatusSink{
			Emit: func(line string) { lines = append(lines, line) },
			Warn: func(message string) { warnings = append(warnings, message) },
		},
	)
	if len(warnings) != 1 || !strings.Contains(warnings[0], "15s") {
		t.Fatalf("warnings = %v", warnings)
	}
	if !strings.Contains(strings.Join(lines, "\n"), "guest cloudflared unknown") {
		t.Fatalf("lines = %v", lines)
	}
}

func stepNames(steps []Step) []string {
	out := make([]string, len(steps))
	for i, s := range steps {
		out[i] = string(s)
	}
	return out
}
