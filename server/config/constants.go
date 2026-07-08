package config

import (
	"os"
	"path/filepath"
)

func resolveDataDir() string {
	if dir := os.Getenv("AI_CRITIC_HOME"); dir != "" {
		return dir
	}
	return ".ai-critic"
}

// DataDir is the base directory for all ai-critic data files, relative to the
// working directory (or under $HOME for per-user configs like agents.json).
var DataDir = resolveDataDir()

// LoopbackHost is used for local TCP/HTTP checks. Prefer 127.0.0.1 over
// "localhost" so remote hosts with flaky DNS/nsswitch do not break health checks.
const LoopbackHost = "127.0.0.1"

// Network ports.
const (
	// DefaultServerPort is the default port for the Go backend server.
	DefaultServerPort = 23712

	// KeepAlivePort is the port for the keep-alive management HTTP server.
	KeepAlivePort = 23312

	// ServerLogFile is the log file for the keep-alive managed server.
	ServerLogFile = "ai-critic-server.log"
)

// File paths relative to DataDir.
var (
	CredentialsFile                = DataDir + "/server-credentials"
	EncKeyFile                     = DataDir + "/enc-key"
	EncKeyPubFile                  = DataDir + "/enc-key.pub"
	DomainsFile                    = DataDir + "/server-domains.json"
	CloudflareFile                 = DataDir + "/cloudflare.json"
	TerminalConfFile               = DataDir + "/terminal-config.json"
	GitUserConfigsFile             = DataDir + "/git-user-configs.json"
	ProjectsFile                   = DataDir + "/projects.json"
	AgentsFile                     = DataDir + "/agents.json"
	OpencodeFile                   = DataDir + "/opencode.json"
	ProjectsDir                    = DataDir + "/projects"
	ServerProjectFile              = DataDir + "/server-project.json"
	AIModelsFile                   = DataDir + "/ai-models.json"
	SSHServerFile                  = DataDir + "/ssh-servers.json"
	OpencodeInternalServerRegistry = DataDir + "/opencode-internal-server.json"
	OpencodeInternalServerLock     = DataDir + "/opencode-internal-server.lock"
	FileTransferDir                = DataDir + "/file-transfer"
)

// Process management directory and paths
var (
	ProcsDir = DataDir + "/procs"
)

func OpencodeInternalServerDir() string {
	return ProcsDir + "/opencode-internal"
}

func OpencodeWebServerDir() string {
	return ProcsDir + "/opencode-web"
}

func BasicAuthProxyDir() string {
	return ProcsDir + "/basic-auth-proxy"
}

func OpencodeInternalServerLockPath() string {
	return filepath.Join(OpencodeInternalServerDir(), "lock")
}

func OpencodeInternalServerRegistryPath() string {
	return filepath.Join(OpencodeInternalServerDir(), "registry.json")
}

func OpencodeWebServerLockPath() string {
	return filepath.Join(OpencodeWebServerDir(), "lock")
}

func OpencodeWebServerRegistryPath() string {
	return filepath.Join(OpencodeWebServerDir(), "registry.json")
}

func BasicAuthProxyLockPath() string {
	return filepath.Join(BasicAuthProxyDir(), "lock")
}

func BasicAuthProxyRegistryPath() string {
	return filepath.Join(BasicAuthProxyDir(), "registry.json")
}
