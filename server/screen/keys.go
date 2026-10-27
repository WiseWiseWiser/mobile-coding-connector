package screen

// modifierKeys are the DOM codes the protocol treats as modifiers: they stick
// on the host and must be released on every disconnect path.
var modifierKeys = map[string]bool{
	"ShiftLeft":    true,
	"ShiftRight":   true,
	"ControlLeft":  true,
	"ControlRight": true,
	"AltLeft":      true,
	"AltRight":     true,
	"MetaLeft":     true,
	"MetaRight":    true,
	"CapsLock":     true,
}

func isModifierKey(code string) bool { return modifierKeys[code] }
