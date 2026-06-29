package agentcli

// Profile selects CLI branding, config path, and local-only behavior.
type Profile struct {
	Name                   string
	ConfigFile             string
	DefaultPort            int
	DefaultServer          string
	SupportsPortFlag       bool
	CheckLocalReachability bool
}

var active Profile

// RemoteProfile is the remote-agent CLI profile.
func RemoteProfile() Profile {
	return Profile{
		Name:                   "remote-agent",
		ConfigFile:             "remote-agent-config.json",
		DefaultPort:            0,
		DefaultServer:          "",
		SupportsPortFlag:       false,
		CheckLocalReachability: false,
	}
}

// LocalProfile is the local-agent CLI profile.
func LocalProfile() Profile {
	return Profile{
		Name:                   "local-agent",
		ConfigFile:             "local-agent-config.json",
		DefaultPort:            23712,
		DefaultServer:          "http://localhost:23712",
		SupportsPortFlag:       true,
		CheckLocalReachability: true,
	}
}