//go:build darwin

package screen

import (
	"fmt"
	"testing"
)

// requiredKeys are the DOM codes the browser side can emit and that the key bar
// offers. Every one of them must be injectable on every platform.
func requiredKeys() []string {
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

func TestDarwinKeyTableCoversRequiredCodes(t *testing.T) {
	for _, code := range requiredKeys() {
		if _, ok := darwinKeyCodes[code]; !ok {
			t.Errorf("darwin key table is missing %s", code)
		}
	}
}

func TestDarwinKeyTableEntriesAreInRange(t *testing.T) {
	for code, vk := range darwinKeyCodes {
		if vk < 0 || vk > 0x7f {
			t.Errorf("%s maps to out-of-range virtual keycode %#x", code, vk)
		}
	}
}

func TestModifierKeysAreKnown(t *testing.T) {
	for code := range modifierKeys {
		if _, ok := darwinKeyCodes[code]; !ok {
			t.Errorf("modifier %s has no darwin keycode", code)
		}
	}
	if !isModifierKey("MetaLeft") || isModifierKey("KeyA") {
		t.Fatal("isModifierKey misbehaves")
	}
}
