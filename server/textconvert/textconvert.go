// Package textconvert provides named text converters for the insert-picker adhoc pad.
package textconvert

import (
	"fmt"
	"strings"
)

const (
	// ConverterShellSingleLine unwraps shell line continuations and collapses whitespace.
	ConverterShellSingleLine = "shell-single-line"
)

// Converter transforms input text.
type Converter func(text string) string

var registry = map[string]Converter{
	ConverterShellSingleLine: ShellSingleLine,
}

// DefaultConverter is used when the request omits a converter id.
const DefaultConverter = ConverterShellSingleLine

// Convert runs the named converter. Empty converterID selects DefaultConverter.
func Convert(text, converterID string) (string, error) {
	id := strings.TrimSpace(converterID)
	if id == "" {
		id = DefaultConverter
	}
	fn, ok := registry[id]
	if !ok {
		return "", fmt.Errorf("unknown converter %q", id)
	}
	return fn(text), nil
}

// KnownConverters returns registered converter ids (stable order for tests).
func KnownConverters() []string {
	return []string{ConverterShellSingleLine}
}

// ShellSingleLine removes backslash-newline continuations, then collapses
// remaining whitespace runs to a single space and trims.
func ShellSingleLine(text string) string {
	unwrapped := unwrapShellContinuations(text)
	return strings.Join(strings.Fields(unwrapped), " ")
}

func unwrapShellContinuations(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		if s[i] == '\\' {
			if i+1 < len(s) && s[i+1] == '\n' {
				i += 2
				continue
			}
			if i+1 < len(s) && s[i+1] == '\r' {
				if i+2 < len(s) && s[i+2] == '\n' {
					i += 3
					continue
				}
				i += 2
				continue
			}
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}
