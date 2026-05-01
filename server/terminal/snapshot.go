package terminal

import (
	"fmt"
	"strings"

	"github.com/hinshun/vt10x"
)

func renderScreenSnapshot(scrollback []byte, cols, rows int) ([]byte, bool) {
	if len(scrollback) == 0 {
		return nil, false
	}
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}

	vt := vt10x.New(vt10x.WithSize(cols, rows))
	if _, err := vt.Write(scrollback); err != nil {
		return nil, false
	}

	vt.Lock()
	defer vt.Unlock()

	var out strings.Builder
	out.Grow(len(scrollback) / 2)
	out.WriteString("\x1b[?25l")
	if vt.Mode()&vt10x.ModeAltScreen != 0 {
		out.WriteString("\x1b[?1049h")
	} else {
		out.WriteString("\x1b[?1049l")
	}
	out.WriteString("\x1b[0m\x1b[H\x1b[2J")

	for y := 0; y < rows; y++ {
		line := renderSnapshotLine(vt, cols, y)
		if line == "" {
			continue
		}
		fmt.Fprintf(&out, "\x1b[%d;1H%s", y+1, line)
	}

	cursor := vt.Cursor()
	cursorX := clamp(cursor.X+1, 1, cols)
	cursorY := clamp(cursor.Y+1, 1, rows)
	fmt.Fprintf(&out, "\x1b[%d;%dH", cursorY, cursorX)
	if vt.CursorVisible() {
		out.WriteString("\x1b[?25h")
	}
	return []byte(out.String()), true
}

func renderSnapshotLine(vt vt10x.Terminal, cols, y int) string {
	runes := make([]rune, cols)
	lastNonSpace := -1
	for x := 0; x < cols; x++ {
		ch := vt.Cell(x, y).Char
		if ch == 0 {
			ch = ' '
		}
		runes[x] = ch
		if ch != ' ' {
			lastNonSpace = x
		}
	}
	if lastNonSpace < 0 {
		return ""
	}
	return string(runes[:lastNonSpace+1])
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
