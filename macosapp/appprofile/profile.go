// Package appprofile defines static local vs remote macOS app product flags.
package appprofile

// Profile holds identity and capability flags for a menu-bar app product.
type Profile struct {
	SpawnsDaemon   bool
	UsesAuthToken  bool
	ConfigFileName string
	BundleID       string
	AppName        string
	DisplayName    string
}

// Local returns the local ai-critic-macos app profile (spawns keep-alive daemon).
func Local() Profile {
	return Profile{
		SpawnsDaemon:   true,
		UsesAuthToken:  false,
		ConfigFileName: "local-agent-config.json",
		BundleID:       "com.xhd2015.ai-critic-macos",
		AppName:        "ai-critic-macos",
		DisplayName:    "AI Critic",
	}
}

// Remote returns the remote ai-critic-remote-macos app profile (no local daemon).
func Remote() Profile {
	return Profile{
		SpawnsDaemon:   false,
		UsesAuthToken:  true,
		ConfigFileName: "remote-agent-config.json",
		BundleID:       "com.xhd2015.ai-critic-remote-macos",
		AppName:        "ai-critic-remote-macos",
		DisplayName:    "AI Critic(Remote)",
	}
}
