package synccmd

import (
	"io"
	"os"

	"golang.org/x/term"
)

// ColorMode selects ANSI color policy (go-best-practice cli/color).
type ColorMode int

const (
	ColorAuto ColorMode = iota
	ColorAlways
	ColorNever
)

const (
	ansiReset  = "\x1b[0m"
	ansiRed    = "\x1b[31m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiGray   = "\x1b[90m"
)

type colorStyle struct{ enabled bool }

func newColorStyle(mode ColorMode, stdout io.Writer) colorStyle {
	return colorStyle{enabled: resolveColor(mode, writerIsTTY(stdout), os.Getenv("NO_COLOR"))}
}

func resolveColor(mode ColorMode, stdoutIsTTY bool, noColorEnv string) bool {
	switch mode {
	case ColorAlways:
		return true
	case ColorNever:
		return false
	default:
		if noColorEnv != "" {
			return false
		}
		return stdoutIsTTY
	}
}

func writerIsTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

func (c colorStyle) wrap(code, s string) string {
	if !c.enabled {
		return s
	}
	return code + s + ansiReset
}

func (c colorStyle) red(s string) string    { return c.wrap(ansiRed, s) }
func (c colorStyle) green(s string) string  { return c.wrap(ansiGreen, s) }
func (c colorStyle) yellow(s string) string { return c.wrap(ansiYellow, s) }
func (c colorStyle) gray(s string) string   { return c.wrap(ansiGray, s) }
