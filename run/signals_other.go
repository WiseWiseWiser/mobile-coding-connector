//go:build !unix

package run

func ignoreJobControlStop() {}
func isManagedServerChild() bool { return false }