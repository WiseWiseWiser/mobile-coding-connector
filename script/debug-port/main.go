package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xhd2015/less-gen/flags"
)

const defaultPort = 5173

const help = `Usage: go run ./script/debug-port [options] [script]

Debug a port using Puppeteer browser automation.

Options:
  -h, --help      Show this help message
  --port PORT     Port to debug (default: 5173)
  --headless      Run in headless mode (default: true)
  --no-headless   Run with visible browser

The script argument is JavaScript code to execute in the browser context.
If no script is provided, reads from stdin.

Example:
  go run ./script/debug-port --port=37651 "console.log(await page.title())"
  echo "await navigate('/'); console.log(await page.title())" | go run ./script/debug-port --port=37651
`

func main() {
	err := run(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	var port int
	var headless bool = true
	var noHeadless bool

	args, err := flags.
		Int("--port", &port).
		Bool("--headless", &headless).
		Bool("--no-headless", &noHeadless).
		Help("-h,--help", help).
		Parse(args)
	if err != nil {
		return err
	}

	if noHeadless {
		headless = false
	}

	if port == 0 {
		port = defaultPort
	}

	var scriptArg string
	if len(args) > 0 {
		scriptArg = args[0]
	}

	projectRoot, err := getProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to get project root: %v", err)
	}

	debugScript := filepath.Join(projectRoot, "script", "debug-port", "debug.js")
	if _, err := os.Stat(debugScript); os.IsNotExist(err) {
		return fmt.Errorf("debug.js not found at %s", debugScript)
	}

	cmd := exec.Command("node", debugScript)
	cmd.Dir = filepath.Join(projectRoot, "script", "debug-port")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Env = append(os.Environ(),
		fmt.Sprintf("BASE_URL=http://localhost:%d", port),
		fmt.Sprintf("HEADLESS=%v", headless),
	)

	if scriptArg != "" {
		cmd.Args = append(cmd.Args, scriptArg)
	}

	return cmd.Run()
}

func getProjectRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
