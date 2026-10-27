//go:build linux

package screen

import (
	"fmt"
	"testing"
)

func requiredX11Keys() []string {
	keys := []string{
		"Tab", "Enter", "Escape", "Backspace", "Space", "CapsLock",
		"ShiftLeft", "ShiftRight", "ControlLeft", "ControlRight",
		"AltLeft", "AltRight", "MetaLeft", "MetaRight",
		"ArrowLeft", "ArrowRight", "ArrowUp", "ArrowDown",
		"Home", "End", "PageUp", "PageDown", "Delete",
		"Minus", "Equal", "BracketLeft", "BracketRight", "Backslash",
		"Semicolon", "Quote", "Comma", "Period", "Slash", "Backquote",
	}
	for c := 'A'; c <= 'Z'; c++ {
		keys = append(keys, "Key"+string(c))
	}
	for i := 0; i <= 9; i++ {
		keys = append(keys, fmt.Sprintf("Digit%d", i))
	}
	for i := 1; i <= 12; i++ {
		keys = append(keys, fmt.Sprintf("F%d", i))
	}
	return keys
}

func TestX11KeysymsCoverRequiredCodes(t *testing.T) {
	for _, code := range requiredX11Keys() {
		if _, ok := x11Keysyms[code]; !ok {
			t.Errorf("x11 keysym table is missing %s", code)
		}
	}
}

func TestX11KeysymsAreNonZero(t *testing.T) {
	for code, sym := range x11Keysyms {
		if sym == 0 {
			t.Errorf("%s maps to keysym 0", code)
		}
	}
}

func TestModifierKeysAreKnownOnX11(t *testing.T) {
	for code := range modifierKeys {
		if _, ok := x11Keysyms[code]; !ok {
			t.Errorf("modifier %s has no keysym", code)
		}
	}
}
