package main

import (
	"embed"
	"fmt"
	"os"

	"github.com/xhd2015/lifelog-private/ai-critic/run"
	"github.com/xhd2015/lifelog-private/ai-critic/server"
)

//go:embed ai-critic-react/dist
var distFS embed.FS

//go:embed ai-critic-react/template.html
var templateHTML string

func main() {
	server.Init(distFS, templateHTML)

	err := run.Run(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
