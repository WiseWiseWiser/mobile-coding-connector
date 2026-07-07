package lib

import (
	"fmt"
	"os"
	"strings"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/agent-pro/agent/commit_msg"
	gitrunner "github.com/xhd2015/agent-pro/agent/git_runner"
)

var genCommitMsgHelp = `Usage: gen-commit-msg [options]

Generate a commit message for the currently staged changes using AI.
Logs are printed to stderr; the resulting commit message is printed to stdout.

Options:
  --dir DIR    Git directory to use (defaults to current directory)
  --model MODEL
              Model to use for generation
  --commit     Run git commit with the generated message after printing it
  -h, --help   Show this help message
`

// RunGenCommitMsg is the shared entry point for the gen-commit-msg CLI.
func RunGenCommitMsg(args []string) error {
	var dir string
	var model string
	var commit bool
	_, err := flags.
		String("--dir", &dir).
		String("--model", &model).
		Bool("--commit", &commit).
		Help("-h,--help", genCommitMsgHelp).
		Parse(args)
	if err != nil {
		return err
	}

	if dir == "" {
		dir, _ = os.Getwd()
	}

	msg, err := commit_msg.Generate(dir, commit_msg.GenerateOptions{
		Model:  model,
		Logger: &stderrLogger{},
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "\n--- Generated Commit Message ---\n")
	fmt.Println(msg)

	quotedMsg := shellQuote(msg)
	fmt.Fprintf(os.Stderr, "\nRun:\n  git commit -m %s\n", quotedMsg)

	if commit {
		fmt.Fprintf(os.Stderr, "\nRunning git commit...\n")
		output, err := gitrunner.Commit(msg, false).Dir(dir).Run()
		if len(output) > 0 {
			fmt.Fprint(os.Stderr, string(output))
		}
		if err != nil {
			return fmt.Errorf("git commit failed: %w", err)
		}
	}

	return nil
}

type stderrLogger struct{}

func (l *stderrLogger) Log(msg string)   { fmt.Fprintf(os.Stderr, "%s\n", msg) }
func (l *stderrLogger) Error(msg string) { fmt.Fprintf(os.Stderr, "ERROR: %s\n", msg) }

func shellQuote(s string) string {
	if !strings.ContainsAny(s, "'\"\\$ !`\n\r\t") {
		return "'" + s + "'"
	}
	return "$'" + strings.NewReplacer(
		"\\", "\\\\",
		"'", "\\'",
		"\n", "\\n",
		"\r", "\\r",
		"\t", "\\t",
	).Replace(s) + "'"
}
