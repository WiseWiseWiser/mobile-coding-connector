package main

import (
	"fmt"
	"os"

	"github.com/xhd2015/ai-critic/cmd/agentcli"
	"github.com/xhd2015/ai-critic/cmd/agentcli/testhooks"
)

func main() {
	testhooks.ApplyFromEnv()
	if err := agentcli.Run(agentcli.LocalProfile(), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}