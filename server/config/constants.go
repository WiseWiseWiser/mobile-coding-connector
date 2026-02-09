package config

// DataDir is the base directory for all ai-critic data files, relative to the
// working directory (or under $HOME for per-user configs like agents.json).
const DataDir = ".ai-critic"

// File paths relative to DataDir.
const (
	CredentialsFile  = DataDir + "/server-credentials"
	EncKeyFile       = DataDir + "/enc-key"
	EncKeyPubFile    = DataDir + "/enc-key.pub"
	DomainsFile      = DataDir + "/server-domains.json"
	CloudflareFile   = DataDir + "/cloudflare.json"
	TerminalConfFile = DataDir + "/terminal-config.json"
	ProjectsFile     = DataDir + "/projects.json"
	AgentsFile       = DataDir + "/agents.json"
	OpencodeFile     = DataDir + "/opencode.json"
	ProjectsDir      = DataDir + "/projects"
)
