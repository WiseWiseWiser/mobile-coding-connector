package main

import (
	"fmt"
	"os"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/lifelog-private/ai-critic/script/lib"
)

const defaultOutput = "ai-critic-server-linux-amd64"

var help = `
Usage: go run ./script/server/build/for-linux-amd64 [options]

Cross-compiles the Go server binary for linux/amd64.

Options:
  -o, --output PATH   Output binary path (default: ` + defaultOutput + `)
  -h, --help          Show this help message
`

func main() {
	err := Handle(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func Handle(args []string) error {
	var output string
	_, err := flags.
		String("-o,--output", &output).
		Help("-h,--help", help).
		Parse(args)
	if err != nil {
		return err
	}

	if output == "" {
		output = defaultOutput
	}

	// Build frontend first so the server embeds the latest dist
	if err := lib.BuildFrontend(); err != nil {
		return err
	}

	return lib.BuildServer(lib.BuildServerOptions{
		Output: output,
		GOOS:   "linux",
		GOARCH: "amd64",
	})
}
