//go:build !unix

package daemon

func stdinIsTerminal() bool {
	return false
}

func resolveEffectiveDetach(explicitDetach bool) bool {
	return explicitDetach || !stdinIsTerminal()
}

func ignoreTerminalHangup() {}

func tryDetachSession() bool {
	return false
}