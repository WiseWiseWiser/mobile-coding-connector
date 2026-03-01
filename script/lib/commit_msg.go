package lib

import (
	"fmt"
	"os"
	"strings"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/lifelog-private/ai-critic/server/agents/commit_msg"
)

var genCommitMsgHelp = `Usage: gen-commit-msg [options]

Generate a commit message for the currently staged changes using AI.
Logs are printed to stderr; the resulting commit message is printed to stdout.

Options:
  --dir DIR    Git directory to use (defaults to current directory)
  -h, --help   Show this help message
`

// RunGenCommitMsg is the shared entry point for the gen-commit-msg CLI.
func RunGenCommitMsg(args []string) error {
	var dir string
	_, err := flags.
		String("--dir", &dir).
		Help("-h,--help", genCommitMsgHelp).
		Parse(args)
	if err != nil {
		return err
	}

	if dir == "" {
		dir, _ = os.Getwd()
	}

	msg, err := commit_msg.Generate(dir, &stderrLogger{})
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "\n--- Generated Commit Message ---\n")
	fmt.Println(msg)

	quotedMsg := shellQuote(msg)
	fmt.Fprintf(os.Stderr, "\nRun:\n  git commit -m %s\n", quotedMsg)

	return nil
}

type stderrLogger struct{}

func (l *stderrLogger) Log(msg string)   { fmt.Fprintf(os.Stderr, "%s\n", msg) }
func (l *stderrLogger) Error(msg string) { fmt.Fprintf(os.Stderr, "ERROR: %s\n", msg) }

func shellQuote(s string) string {
	s = strings.ReplaceAll(s, "\n", "\\n")
	if !strings.ContainsAny(s, "'\"\\$ !`") {
		return "'" + s + "'"
	}
	return "$'" + strings.NewReplacer("\\", "\\\\", "'", "\\'", "\n", "\\n").Replace(s) + "'"
}
