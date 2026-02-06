package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/xgo/support/cmd"
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

	fmt.Printf("Building Go server -> %s\n", output)
	err = cmd.Debug().Run("go", "build", "-o", output, "./")
	if err != nil {
		return fmt.Errorf("failed to build Go server: %v", err)
	}

	fmt.Printf("Server binary built: %s\n", output)
	return nil
}
