package agentcli

import (
	"bufio"
	"os"
	"time"
)

const escFollowTimeout = 40 * time.Millisecond

// readTerminalsKey reads one logical key from in.
// Arrow keys are CSI sequences (ESC [ A/B/C/D). A lone ESC is "esc":
// after ESC, a TTY file uses a short deadline; a buffer (tests) hits EOF.
func readTerminalsKey(in *bufio.Reader, tty *os.File) (string, error) {
	b, err := in.ReadByte()
	if err != nil {
		return "", err
	}
	if b != 0x1b {
		return mapTerminalsKeyByte(b), nil
	}
	if in.Buffered() == 0 && tty != nil {
		_ = tty.SetReadDeadline(time.Now().Add(escFollowTimeout))
		defer func() { _ = tty.SetReadDeadline(time.Time{}) }()
	}
	next, err := in.ReadByte()
	if err != nil {
		return "esc", nil
	}
	if next != '[' {
		if k := mapTerminalsKeyByte(next); k != "" && k != "esc" {
			return k, nil
		}
		return "esc", nil
	}
	code, err := in.ReadByte()
	if err != nil {
		return "esc", nil
	}
	switch code {
	case 'A':
		return "up", nil
	case 'B':
		return "down", nil
	case 'C':
		return "right", nil
	case 'D':
		return "left", nil
	case 'Z':
		return "tab", nil
	default:
		return "", nil
	}
}

func mapTerminalsKeyByte(b byte) string {
	switch b {
	case '\r', '\n':
		return "enter"
	case '\t':
		return "tab"
	case 0x1b:
		return "esc"
	case 0x03: // Ctrl-C in raw mode
		return "q"
	case 'q', 'Q':
		return "q"
	default:
		if b >= 32 && b < 127 {
			return string(b)
		}
		return ""
	}
}
