package launchenv

// KeepAliveEnv builds child-process environment for ai-critic keep-alive spawned
// from the macOS menu-bar app: suppress browser auto-open only (usage fetch is in-process).
func KeepAliveEnv(binaryDir string) map[string]string {
	_ = binaryDir
	return map[string]string{
		"AI_CRITIC_NO_OPEN_BROWSER": "1",
	}
}