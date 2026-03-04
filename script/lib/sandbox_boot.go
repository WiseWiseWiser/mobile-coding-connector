package lib

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/xhd2015/less-gen/flags"
)

// SandboxCLIParsed holds the result of ParseSandboxCLI.
type SandboxCLIParsed struct {
	ArchFlag string
}

// ParseSandboxCLI parses common sandbox CLI flags (--arch, --reset)
// and handles the container reset flow. Returns nil (with no error) if
// the user aborted the reset prompt.
func ParseSandboxCLI(args []string, help string, containerName string) (*SandboxCLIParsed, error) {
	var archFlag string
	var reset bool
	_, err := flags.
		String("--arch", &archFlag).
		Bool("--reset", &reset).
		Help("-h,--help", help).
		Parse(args)
	if err != nil {
		return nil, err
	}
	if archFlag == "" {
		archFlag = "auto"
	}

	if reset {
		if _, err := InspectContainerStatus(containerName); err == nil {
			if IsStdinTTY() {
				fmt.Printf("Container %q exists. Destroy and recreate? [y/N] ", containerName)
				reader := bufio.NewReader(os.Stdin)
				answer, _ := reader.ReadString('\n')
				if strings.TrimSpace(strings.ToLower(answer)) != "y" {
					fmt.Println("Aborted.")
					return nil, nil
				}
			}
			fmt.Println("Resetting: removing existing container...")
			_ = RunVerbose("podman", "rm", "-f", containerName)
		}
	}

	return &SandboxCLIParsed{ArchFlag: archFlag}, nil
}

// SandboxBootOptions configures RunSandboxBoot.
type SandboxBootOptions struct {
	Help    string
	Sandbox SandboxOptions
}

// RunSandboxBoot parses common sandbox CLI flags (--arch, --reset),
// handles container reset, and runs the sandbox.
func RunSandboxBoot(args []string, opts SandboxBootOptions) error {
	parsed, err := ParseSandboxCLI(args, opts.Help, opts.Sandbox.ContainerName)
	if err != nil {
		return err
	}
	if parsed == nil {
		return nil
	}
	opts.Sandbox.ArchFlag = parsed.ArchFlag
	return RunSandbox(opts.Sandbox)
}

func IsStdinTTY() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}
