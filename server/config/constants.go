package config

// DataDir is the base directory for all ai-critic data files, relative to the
// working directory (or under $HOME for per-user configs like agents.json).
const DataDir = ".ai-critic"

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
const (
	CredentialsFile   = DataDir + "/server-credentials"
	EncKeyFile        = DataDir + "/enc-key"
	EncKeyPubFile     = DataDir + "/enc-key.pub"
	DomainsFile       = DataDir + "/server-domains.json"
	CloudflareFile    = DataDir + "/cloudflare.json"
	TerminalConfFile  = DataDir + "/terminal-config.json"
	ProjectsFile      = DataDir + "/projects.json"
	AgentsFile        = DataDir + "/agents.json"
	OpencodeFile      = DataDir + "/opencode.json"
	ProjectsDir       = DataDir + "/projects"
	ServerProjectFile = DataDir + "/server-project.json"
	AIModelsFile      = DataDir + "/ai-models.json"
)
