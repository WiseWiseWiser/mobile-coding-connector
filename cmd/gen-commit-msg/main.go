package main

import (
	"fmt"
	"os"

	"github.com/xhd2015/lifelog-private/ai-critic/script/lib"
)

func main() {
	if err := lib.RunGenCommitMsg(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
