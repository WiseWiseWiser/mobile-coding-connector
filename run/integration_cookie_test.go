package run

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhd2015/ai-critic/server/auth"
)

func setupCookieSystemctlFixture(t *testing.T) (*bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	resetIntegrationHooks()
	t.Cleanup(resetIntegrationHooks)
	integrationUnitPath = filepath.Join(t.TempDir(), "ai-critic-server.service")
	integrationExecutable = func() (string, error) { return "/opt/ai-critic-server", nil }
	integrationGetuid = func() int { return 0 }
	integrationSystemctl = func(args ...string) (string, error) { return "active\n", nil }
	out, errOut := new(bytes.Buffer), new(bytes.Buffer)
	integrationOut, integrationErr = out, errOut
	return out, errOut
}

func TestSystemctlCookieNameInUnitAndCommands(t *testing.T) {
	out, _ := setupCookieSystemctlFixture(t)
	if err := Run([]string{"integration", "systemctl", "start", "--auth-cookie-name", "pi-agent-token"}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(integrationUnitPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `ExecStart=/opt/ai-critic-server --port 23712 --auth-cookie-name "pi-agent-token"`) {
		t.Fatalf("unit missing configured cookie: %s", body)
	}
	out.Reset()
	if err := Run([]string{"integration", "systemctl", "show-config", "--auth-cookie-name", "pi-agent-token"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "on-disk:    match") {
		t.Fatal("same explicit cookie configuration no longer matches")
	}
	for _, line := range strings.Split(out.String(), "\n") {
		if strings.Contains(line, " integration systemctl start") && !strings.Contains(line, "--auth-cookie-name") {
			t.Fatalf("printed maintenance command loses configuration: %s", line)
		}
	}
}

func TestSystemctlCookieNameCannotBeSilentlyReset(t *testing.T) {
	for _, dry := range []bool{false, true} {
		t.Run(map[bool]string{true: "dry-run", false: "write"}[dry], func(t *testing.T) {
			out, errOut := setupCookieSystemctlFixture(t)
			original := generateSystemdUnit("/opt/ai-critic-server", "/opt", "/root", 23712, "pi-agent-token")
			if err := os.WriteFile(integrationUnitPath, []byte(original), 0600); err != nil {
				t.Fatal(err)
			}
			integrationSystemctl = func(args ...string) (string, error) {
				t.Fatal("systemctl called before protecting existing cookie option")
				return "", nil
			}
			args := []string{"integration", "systemctl", "start"}
			if dry {
				args = append(args, "--dry-run")
			}
			if err := Run(args); err == nil || !strings.Contains(err.Error(), "repeat the option explicitly") {
				t.Fatalf("unsafe reset allowed: %v", err)
			}
			body, _ := os.ReadFile(integrationUnitPath)
			if string(body) != original {
				t.Fatal("existing configuration was overwritten")
			}
			if err := Run([]string{"integration", "systemctl", "show-config"}); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(errOut.String(), "warning:") || !strings.Contains(out.String(), "on-disk:    differ") {
				t.Fatal("default preview must warn without changing custom configuration")
			}
		})
	}
}

func TestSystemctlCookieNameFlagPlacementAndValidation(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
		fail bool
	}{
		{"before", []string{"--auth-cookie-name", "pi-agent-token", "start", "--dry-run"}, "pi-agent-token", false},
		{"after", []string{"start", "--dry-run", "--auth-cookie-name", "pi-agent-token"}, "pi-agent-token", false},
		{"subcommand-override", []string{"--auth-cookie-name", "pi-agent-token", "start", "--dry-run", "--auth-cookie-name", "other-agent-token"}, "other-agent-token", false},
		{"empty", []string{"start", "--dry-run", "--auth-cookie-name", ""}, "invalid --auth-cookie-name", true},
		{"reserved", []string{"start", "--auth-cookie-name", auth.ServiceCookieName}, "invalid --auth-cookie-name", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, _ := setupCookieSystemctlFixture(t)
			integrationSystemctl = func(args ...string) (string, error) { t.Fatal("validation/dry-run invoked systemctl"); return "", nil }
			args := append([]string{"integration", "systemctl"}, tc.args...)
			if !tc.fail { // show-config is the read-only producer for inspecting the exact unit.
				for i, arg := range args {
					if arg == "start" {
						args[i] = "show-config"
					}
				}
			}
			err := Run(args)
			if tc.fail {
				if err == nil || !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("expected validation error: %v", err)
				}
				if _, err := os.Stat(integrationUnitPath); !os.IsNotExist(err) {
					t.Fatal("invalid input created a service definition")
				}
			} else if err != nil || !strings.Contains(out.String(), tc.want) {
				t.Fatalf("wrong cookie option: %v, %s", err, out.String())
			}
		})
	}
}

func TestSystemctlCookieArgumentEscapingAndExplicitDefault(t *testing.T) {
	unit := generateSystemdUnit("/opt/server", "/opt", "/root", 23712, "pi-$%token")
	if !strings.Contains(unit, `--auth-cookie-name "pi-$$%%token"`) {
		t.Fatalf("systemd would expand cookie argument: %s", unit)
	}
	out, _ := setupCookieSystemctlFixture(t)
	if err := os.WriteFile(integrationUnitPath, []byte(generateSystemdUnit("/opt/ai-critic-server", "/opt", "/root", 23712, "pi-agent-token")), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"integration", "systemctl", "start", "--auth-cookie-name", auth.DefaultCookieName}); err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(integrationUnitPath)
	if strings.Contains(string(body), "--auth-cookie-name") || !strings.Contains(out.String(), "updated") {
		t.Fatal("explicit reset to default did not work")
	}
}
