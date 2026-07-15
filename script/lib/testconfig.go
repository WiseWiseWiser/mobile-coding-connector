package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	serverenv "github.com/xhd2015/ai-critic/server/env"
)

const (
	EnvAI_CRITIC_HOME = "AI_CRITIC_HOME"

	TestUsername = "test"
	TestPassword = "testpassword"
)

// CreateTestConfigHome creates an isolated temporary config directory for tests.
func CreateTestConfigHome() (string, error) {
	return os.MkdirTemp("", "ai-critic-test-*")
}

// TestCredentialsFile returns the server-credentials path under a config home.
func TestCredentialsFile(configHome string) string {
	return filepath.Join(configHome, "server-credentials")
}

// WriteTestCredentials writes the default test password token into config home.
func WriteTestCredentials(configHome string) (string, error) {
	credFile := TestCredentialsFile(configHome)
	if err := os.WriteFile(credFile, []byte(TestPassword+"\n"), 0600); err != nil {
		return "", fmt.Errorf("failed to write test credentials: %w", err)
	}
	return credFile, nil
}

// IsUnderTempDir reports whether path is inside os.TempDir().
func IsUnderTempDir(path string) bool {
	tempDir, err := filepath.Abs(os.TempDir())
	if err != nil {
		return false
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(tempDir, abs)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func (o *QuickTestOptions) ensureConfigHome() (credFile string, err error) {
	if o.Local {
		return "", nil
	}

	if o.ConfigHome == "" {
		o.ConfigHome, err = CreateTestConfigHome()
		if err != nil {
			return "", err
		}
		o.managedConfigHome = true
	}

	credFile, err = WriteTestCredentials(o.ConfigHome)
	if err != nil {
		if o.managedConfigHome {
			os.RemoveAll(o.ConfigHome)
			o.ConfigHome = ""
			o.managedConfigHome = false
		}
		return "", err
	}
	return credFile, nil
}

// QuickTestCleanup removes a managed temporary config home created by QuickTestStart.
func QuickTestCleanup(opts *QuickTestOptions) {
	if opts == nil || !opts.managedConfigHome || opts.ConfigHome == "" {
		return
	}
	os.RemoveAll(opts.ConfigHome)
	opts.ConfigHome = ""
	opts.managedConfigHome = false
}

func appendQuickTestServerEnv(base []string, configHome string) []string {
	if configHome == "" {
		return append(base, envNoOpenBrowser())
	}

	env := make([]string, 0, len(base)+2)
	for _, e := range base {
		if strings.HasPrefix(e, EnvAI_CRITIC_HOME+"=") {
			continue
		}
		env = append(env, e)
	}
	env = append(env, EnvAI_CRITIC_HOME+"="+configHome)
	return append(env, envNoOpenBrowser())
}

func envNoOpenBrowser() string {
	return serverenv.EnvNoOpenBrowser + "=1"
}

// AppendTestServerEnv returns env for a doctest server process: isolated config
// home and no auto-open browser.
func AppendTestServerEnv(base []string, configHome string) []string {
	env := make([]string, 0, len(base)+2)
	for _, e := range base {
		if strings.HasPrefix(e, EnvAI_CRITIC_HOME+"=") {
			continue
		}
		if strings.HasPrefix(e, serverenv.EnvNoOpenBrowser+"=") {
			continue
		}
		env = append(env, e)
	}
	env = append(env, EnvAI_CRITIC_HOME+"="+configHome)
	env = append(env, envNoOpenBrowser())
	env = append(env, "AI_CRITIC_TEST_SKIP_EXTENSION=1")
	return env
}