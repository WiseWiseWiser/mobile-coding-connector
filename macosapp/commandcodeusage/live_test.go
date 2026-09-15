package commandcodeusage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLiveSandboxHomes fetches real Command Code accounts. It is opt-in because
// it reads the developer's own credentials and calls the provider API:
//
//	AI_CRITIC_LIVE_COMMANDCODE_HOMES=~/.sandbox/commandcode-v1/.commandcode,~/.sandbox/commandcode-v2/.commandcode go test ./macosapp/commandcodeusage/ -run TestLiveSandboxHomes -v
func TestLiveSandboxHomes(t *testing.T) {
	spec := os.Getenv("AI_CRITIC_LIVE_COMMANDCODE_HOMES")
	if spec == "" {
		t.Skip("set AI_CRITIC_LIVE_COMMANDCODE_HOMES to fetch real accounts")
	}
	for _, raw := range strings.Split(spec, ",") {
		home := expandHome(strings.TrimSpace(raw))
		if home == "" {
			continue
		}
		t.Run(filepath.Base(filepath.Dir(home)), func(t *testing.T) {
			svc := TestExported_NewLiveService(home, "")
			resp := TestExported_FetchOnce(t, svc)
			if resp.Status != StatusReady {
				t.Fatalf("status = %q, error = %q", resp.Status, resp.Error)
			}
			t.Logf("title=%q body=%q url=%q", TitleSuffix(resp), FormatBody(resp), resp.UsageURL)
			if resp.PlanName == "" {
				t.Fatal("ready response must name the plan")
			}
			if TitleSuffix(resp) == "" {
				t.Fatal("ready response must render a menu-bar percent")
			}
		})
	}
}

func expandHome(path string) string {
	if len(path) > 1 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		return filepath.Join(home, path[2:])
	}
	return path
}
