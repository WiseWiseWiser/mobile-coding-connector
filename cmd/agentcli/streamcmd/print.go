package streamcmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/xhd2015/ai-critic/client"
)

func flushStdout() {
	_ = os.Stdout.Sync()
}

// FlushStdout flushes stdout after incremental CLI output.
func FlushStdout() {
	flushStdout()
}

func statusTag(status string) string {
	switch status {
	case "ok":
		return "[ok]"
	case "warn":
		return "[warn]"
	case "skip":
		return "[skip]"
	default:
		return "[fail]"
	}
}

// DefaultLog prints a log event with the builtin indent.
func DefaultLog(ev client.StreamEvent) error {
	fmt.Printf("  %s\n", ev.Message)
	flushStdout()
	return nil
}

// DefaultSection prints a section title.
func DefaultSection(ev client.StreamEvent) error {
	fmt.Printf("%s:\n", ev.Message)
	flushStdout()
	return nil
}

// DefaultProgress prints a progress check line.
func DefaultProgress(ev client.StreamEvent) error {
	return PrintProgress(ev)
}

// PrintProgress formats and prints one progress check (exported for After hooks).
func PrintProgress(ev client.StreamEvent) error {
	line := fmt.Sprintf("%s %s", statusTag(ev.Status), ev.Name)
	if ev.Detail != "" {
		line += ": " + ev.Detail
	}
	fmt.Println(line)
	if ev.Hint != "" {
		fmt.Printf("         hint:\n%s\n", indentHint(ev.Hint))
	}
	flushStdout()
	return nil
}

// DefaultMeta prints meta banner lines for doctor-style streams.
func DefaultMeta(ev client.StreamEvent) error {
	if ev.Message != "" {
		fmt.Println(ev.Message)
	}
	if ev.TryURL != "" {
		fmt.Printf("Try URL: %s\n", ev.TryURL)
	}
	if ev.ServerStatus != nil {
		s := ev.ServerStatus
		fmt.Printf("Server status: running=%v public_url=%v port=%v temporary=%v\n",
			s["running"], s["public_url"], s["port"], s["is_tmp"])
	}
	flushStdout()
	return nil
}

func indentHint(hint string) string {
	lines := strings.Split(strings.TrimSpace(hint), "\n")
	for i, line := range lines {
		lines[i] = "           " + line
	}
	return strings.Join(lines, "\n")
}