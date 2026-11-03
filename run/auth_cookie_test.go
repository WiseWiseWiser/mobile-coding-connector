package run

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhd2015/ai-critic/server/auth"
)

func TestResolveAuthCookieName(t *testing.T) {
	name, err := resolveAuthCookieName(nil)
	if err != nil || name != auth.DefaultCookieName {
		t.Fatalf("default: %q, %v", name, err)
	}
	for _, value := range []string{"pi-agent-token", "", "bad name", "svc-auth-token", "__Host-token"} {
		got, err := resolveAuthCookieName(&value)
		if value == "pi-agent-token" {
			if err != nil || got != value {
				t.Fatalf("custom: %q, %v", got, err)
			}
		} else if err == nil || !strings.Contains(err.Error(), "--auth-cookie-name") {
			t.Fatalf("invalid %q: %v", value, err)
		}
	}
}

func TestReportAuthCookieNameDoesNotPrintCredentials(t *testing.T) {
	var out, errOut bytes.Buffer
	reportAuthCookieName(&out, &errOut, "pi-agent-token")
	if out.String() != "auth cookie name: pi-agent-token\n" {
		t.Fatalf("stdout: %q", out.String())
	}
	if !strings.HasPrefix(errOut.String(), "warning:") || !strings.Contains(errOut.String(), "sign in again") {
		t.Fatalf("stderr: %q", errOut.String())
	}
	out.Reset()
	errOut.Reset()
	reportAuthCookieName(&out, &errOut, auth.DefaultCookieName)
	if errOut.Len() != 0 {
		t.Fatal("default setting should not warn")
	}
}

// Existing Run/flags help paths use process stdio and os.Exit. A test executable
// subprocess isolates those contracts without swapping global stdio or HOME.
func TestAuthCookieCLIHelper(t *testing.T) {
	if os.Getenv("AI_CRITIC_AUTH_COOKIE_CLI_HELPER") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			if err := Run(os.Args[i+1:]); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			os.Exit(0)
		}
	}
	os.Exit(2)
}

func TestAuthCookieCLIPaths(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
		fail bool
	}{
		{"root-help", []string{"--help"}, "--auth-cookie-name", false},
		{"keep-alive-help", []string{"keep-alive", "-h"}, "--auth-cookie-name", false},
		{"systemctl-help", []string{"integration", "systemctl", "--help"}, "--auth-cookie-name", false},
		{"start-help", []string{"integration", "systemctl", "start", "--help"}, "--auth-cookie-name", false},
		{"show-config-help", []string{"integration", "systemctl", "show-config", "-h"}, "--auth-cookie-name", false},
		{"root-empty-rejected", []string{"--auth-cookie-name", ""}, "invalid --auth-cookie-name", true},
		{"root-space-rejected", []string{"--auth-cookie-name", "bad name"}, "invalid --auth-cookie-name", true},
		{"keep-alive-empty-rejected", []string{"keep-alive", "--auth-cookie-name", "", "--script"}, "invalid --auth-cookie-name", true},
		{"systemctl-empty-rejected-before-os", []string{"integration", "systemctl", "start", "--auth-cookie-name", "", "--dry-run"}, "invalid --auth-cookie-name", true},
		{"root-custom-configures-before-component-list", []string{"--auth-cookie-name", "pi-agent-token", "--component", "list"}, "Available components", false},
		{"keep-alive-script-forwards-custom", []string{"keep-alive", "--script", "--port", "49191", "--auth-cookie-name", "pi-agent-token"}, "pi-agent-token", false},
		{"keep-alive-script-preserves-default", []string{"keep-alive", "--script", "--port", "49191"}, "49191", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			home := t.TempDir()
			cmd := exec.Command(os.Args[0], append([]string{"-test.run=^TestAuthCookieCLIHelper$", "--"}, tc.args...)...)
			cmd.Env = append(os.Environ(), "AI_CRITIC_AUTH_COOKIE_CLI_HELPER=1", "HOME="+home, "XDG_CONFIG_HOME="+filepath.Join(home, ".config"))
			var out, errOut bytes.Buffer
			cmd.Stdout, cmd.Stderr = &out, &errOut
			err := cmd.Run()
			if (err != nil) != tc.fail {
				t.Fatalf("err=%v stdout=%s stderr=%s", err, out.String(), errOut.String())
			}
			combined := out.String() + errOut.String()
			if !strings.Contains(combined, tc.want) {
				t.Fatalf("missing %q: %s", tc.want, combined)
			}
			if tc.fail && (out.Len() != 0 || !strings.HasPrefix(errOut.String(), "Error:")) {
				t.Fatal("validation errors must use stderr before startup")
			}
			if tc.name == "keep-alive-script-forwards-custom" {
				commands := 0
				for _, line := range strings.Split(out.String(), "\n") {
					if strings.Contains(line, "--auth-cookie-name") {
						commands++
						if strings.Count(line, "--auth-cookie-name") != 1 || !strings.Contains(line, "pi-agent-token") {
							t.Fatal("cookie name flag lost or duplicated within a server command")
						}
					}
				}
				if commands != 2 {
					t.Fatal("both tee and direct server branches must preserve the cookie option")
				}
			}
			if tc.name == "keep-alive-script-preserves-default" && strings.Contains(out.String(), "--auth-cookie-name") {
				t.Fatal("omitted option changed default child argv")
			}
		})
	}
}
