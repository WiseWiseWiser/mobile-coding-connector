package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/xhd2015/ai-critic/script/lib"
	"github.com/xhd2015/less-gen/flags"
)

var help = `
Usage: go run ./script/server/build [options]

Builds the Go server binary.

Options:
  -o, --output PATH   Output binary path (default: /tmp/ai-critic)
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
		output = "/tmp/ai-critic"
		if runtime.GOOS == "windows" {
			output = "/tmp/ai-critic.exe"
		}
	}

	return lib.BuildServer(lib.BuildServerOptions{
		Output: output,
	})
}
