package terminal

import "github.com/xhd2015/dot-pkgs/go-pkgs/shell/ptywrap"

// ShellQuote preserves the terminal package API used by run scripts.
func ShellQuote(s string) string {
	return ptywrap.ShellQuote(s)
}
