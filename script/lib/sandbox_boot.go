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
	DevMode  bool
}

// ParseSandboxCLI parses common sandbox CLI flags (--arch, --recreate-container,
// --force-recreate-container, --dev) and handles the container recreation flow.
// Returns nil (with no error) if the user aborted the prompt.
func ParseSandboxCLI(args []string, help string, containerName string) (*SandboxCLIParsed, error) {
	var archFlag string
	var recreate bool
	var forceRecreate bool
	var devMode bool
	_, err := flags.
		String("--arch", &archFlag).
		Bool("--recreate-container", &recreate).
		Bool("--force-recreate-container", &forceRecreate).
		Bool("--dev", &devMode).
		Help("-h,--help", help).
		Parse(args)
	if err != nil {
		return nil, err
	}
	if archFlag == "" {
		archFlag = "auto"
	}

	if forceRecreate {
		recreate = true
	}

	if recreate {
		if _, err := InspectContainerStatus(containerName); err == nil {
			if !forceRecreate && IsStdinTTY() {
				fmt.Printf("Container %q exists. Destroy and recreate? [y/N] ", containerName)
				reader := bufio.NewReader(os.Stdin)
				answer, _ := reader.ReadString('\n')
				if strings.TrimSpace(strings.ToLower(answer)) != "y" {
					fmt.Println("Aborted.")
					return nil, nil
				}
			}
			fmt.Println("Removing existing container...")
			_ = RunVerbose("podman", "rm", "-f", containerName)
		}
	}

	return &SandboxCLIParsed{ArchFlag: archFlag, DevMode: devMode}, nil
}

// SandboxBootOptions configures RunSandboxBoot.
type SandboxBootOptions struct {
	Help    string
	Sandbox SandboxOptions
}

// RunSandboxBoot parses common sandbox CLI flags (--arch, --recreate-container),
// handles container recreation, and runs the sandbox.
func RunSandboxBoot(args []string, opts SandboxBootOptions) error {
	parsed, err := ParseSandboxCLI(args, opts.Help, opts.Sandbox.ContainerName)
	if err != nil {
		return err
	}
	if parsed == nil {
		return nil
	}
	opts.Sandbox.ArchFlag = parsed.ArchFlag
	opts.Sandbox.DevMode = parsed.DevMode
	return RunSandbox(opts.Sandbox)
}

func IsStdinTTY() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}
