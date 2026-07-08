//go:build unix

package daemon

import "testing"

func TestResolveEffectiveDetachExplicit(t *testing.T) {
	if !resolveEffectiveDetach(true) {
		t.Fatal("explicit --detach must enable detach")
	}
}

func TestResolveEffectiveDetachNonTTYStdin(t *testing.T) {
	// keep-alive doctest/CI runs with stdin piped or redirected.
	if !resolveEffectiveDetach(false) {
		t.Fatal("non-terminal stdin must auto-detach even without --detach")
	}
}