package main

import (
	"fmt"
	"strings"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/lifelog-private/ai-critic/client"
)

const serviceHelp = `Usage: remote-agent service <subcommand> [args...]

Manage remote services configured in the frontend's Services tab.

Subcommands:
  list [--project-dir <dir>]
      List managed services visible to the remote server.

  start <service-name-or-id>
      Start one service.

  stop <service-name-or-id>
      Stop one service.

  restart <service-name-or-id>
      Restart one service.

  upgrade <service-name-or-id> <local-binary> [--target <remote-path>]
      Upload a replacement binary, then stop, replace, and start one service.

  logs [--lines N] <service-name-or-id>
      Stream one service's log file.
`

const serviceListHelp = `Usage: remote-agent service list [--project-dir <dir>]

List services from the remote server.

Options:
  --project-dir DIR   Filter to the same project scope used by the frontend.
  -h, --help          Show this help message.
`

const serviceLogsHelp = `Usage: remote-agent service logs [--lines N] <service-name-or-id>

Stream logs for one managed service using the same backend log stream as
the frontend.

Options:
  --lines N           Initial tail size before following new log lines.
                      Defaults to 100.
  -h, --help          Show this help message.
`

func runService(resolve func() (*client.Client, error), args []string) error {
	if len(args) == 0 {
		fmt.Print(serviceHelp)
		return nil
	}

	switch args[0] {
	case "list":
		return runServiceList(resolve, args[1:])
	case "start":
		return runServiceAction(resolve, "start", args[1:])
	case "stop":
		return runServiceAction(resolve, "stop", args[1:])
	case "restart":
		return runServiceAction(resolve, "restart", args[1:])
	case "upgrade":
		return runServiceUpgrade(resolve, args[1:])
	case "logs":
		return runServiceLogs(resolve, args[1:])
	case "-h", "--help":
		fmt.Print(serviceHelp)
		return nil
	default:
		return fmt.Errorf("unknown service subcommand: %s", args[0])
	}
}

func runServiceList(resolve func() (*client.Client, error), args []string) error {
	var projectDir string
	args, err := flags.
		String("--project-dir", &projectDir).
		Help("-h,--help", serviceListHelp).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) > 0 {
		return fmt.Errorf("service list takes no positional arguments, got %v", args)
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	services, err := cli.ListServices(projectDir)
	if err != nil {
		return err
	}
	if len(services) == 0 {
		if strings.TrimSpace(projectDir) == "" {
			fmt.Println("No services found.")
		} else {
			fmt.Printf("No services found for project scope %q.\n", projectDir)
		}
		return nil
	}

	for i, service := range services {
		if i > 0 {
			fmt.Println()
		}
		printService(service)
	}
	return nil
}

func runServiceAction(resolve func() (*client.Client, error), action string, args []string) error {
	if len(args) != 1 {
		if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
			fmt.Printf("Usage: remote-agent service %s <service-name-or-id>\n", action)
			return nil
		}
		return fmt.Errorf("service %s requires exactly 1 argument <service-name-or-id>", action)
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	service, err := resolveServiceTarget(cli, args[0])
	if err != nil {
		return err
	}

	switch action {
	case "start":
		updated, err := cli.StartService(service.ID)
		if err != nil {
			return err
		}
		fmt.Printf("Started service %s (%s)\n", updated.ID, displayOrDash(updated.Name))
		fmt.Printf("Status: %s  PID: %s\n", displayOrDash(updated.Status), formatOptionalInt(updated.PID))
	case "stop":
		if err := cli.StopService(service.ID); err != nil {
			return err
		}
		fmt.Printf("Stopped service %s (%s)\n", service.ID, displayOrDash(service.Name))
	case "restart":
		if err := cli.RestartService(service.ID); err != nil {
			return err
		}
		fmt.Printf("Restarted service %s (%s)\n", service.ID, displayOrDash(service.Name))
	default:
		return fmt.Errorf("unsupported service action: %s", action)
	}
	return nil
}

func runServiceLogs(resolve func() (*client.Client, error), args []string) error {
	lines := 100
	args, err := flags.
		Int("--lines", &lines).
		Help("-h,--help", serviceLogsHelp).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) != 1 {
		return fmt.Errorf("service logs requires exactly 1 argument <service-name-or-id>")
	}
	if lines <= 0 {
		return fmt.Errorf("--lines must be greater than 0")
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	service, err := resolveServiceTarget(cli, args[0])
	if err != nil {
		return err
	}
	if strings.TrimSpace(service.LogPath) == "" {
		return fmt.Errorf("service %s (%s) does not have a log path", service.ID, displayOrDash(service.Name))
	}

	fmt.Printf("Streaming logs for %s (%s)\n", displayOrDash(service.Name), service.ID)
	fmt.Printf("Log path: %s\n", service.LogPath)
	fmt.Println("Press Ctrl+C to stop.")

	return cli.StreamLogFile(service.LogPath, lines, func(ev client.LogStreamEvent) {
		switch ev.Type {
		case "log":
			if ev.Message != "" {
				fmt.Println(ev.Message)
			}
		case "status":
			if ev.Message != "" {
				fmt.Println(ev.Message)
			} else if ev.Status != "" {
				fmt.Println(ev.Status)
			}
		}
	})
}

func resolveServiceTarget(cli *client.Client, idOrName string) (*client.ServiceStatus, error) {
	services, err := cli.ListServices("")
	if err != nil {
		return nil, err
	}
	return matchServiceTarget(services, idOrName)
}

func matchServiceTarget(services []client.ServiceStatus, idOrName string) (*client.ServiceStatus, error) {
	idOrName = strings.TrimSpace(idOrName)
	if idOrName == "" {
		return nil, fmt.Errorf("service target cannot be empty")
	}

	for _, service := range services {
		if service.ID == idOrName {
			service := service
			return &service, nil
		}
	}

	var matches []client.ServiceStatus
	for _, service := range services {
		if service.Name == idOrName {
			matches = append(matches, service)
		}
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no service found for %q", idOrName)
	case 1:
		return &matches[0], nil
	default:
		ids := make([]string, 0, len(matches))
		for _, match := range matches {
			ids = append(ids, match.ID)
		}
		return nil, fmt.Errorf("service name %q is ambiguous; matching IDs: %s", idOrName, strings.Join(ids, ", "))
	}
}

func printService(service client.ServiceStatus) {
	const labelWidth = 12
	label := func(name string) string {
		return fmt.Sprintf("  %-*s", labelWidth, name+":")
	}

	fmt.Printf("%s %s\n", label("Name"), displayOrDash(service.Name))
	fmt.Printf("%s %s\n", label("ID"), service.ID)
	fmt.Printf("%s %s\n", label("Status"), displayOrDash(service.Status))
	fmt.Printf("%s %s\n", label("PID"), formatOptionalInt(service.PID))
	fmt.Printf("%s %s\n", label("Scope"), serviceScope(service))
	fmt.Printf("%s %s\n", label("Work Dir"), displayOrDash(service.WorkingDir))
	fmt.Printf("%s %s\n", label("Command"), displayOrDash(service.Command))
	fmt.Printf("%s %s\n", label("Desired"), boolWord(service.DesiredRunning))
	fmt.Printf("%s %s\n", label("Log Path"), displayOrDash(service.LogPath))
	if service.UpgradeTarget != "" {
		fmt.Printf("%s %s\n", label("Upgrade"), service.UpgradeTarget)
	}

	if service.PortForward != nil {
		fmt.Printf("%s %s\n", label("Port"), formatPortForward(service.PortForward))
	}
	if service.LastStartedAt != "" {
		fmt.Printf("%s %s\n", label("Started"), formatAgentTime(service.LastStartedAt))
	}
	if service.LastExitedAt != "" {
		fmt.Printf("%s %s\n", label("Exited"), formatAgentTime(service.LastExitedAt))
	}
	if service.LastExitError != "" {
		fmt.Printf("%s %s\n", label("Last Error"), service.LastExitError)
	}
}

func serviceScope(service client.ServiceStatus) string {
	if strings.TrimSpace(service.ProjectDir) == "" {
		return "all projects"
	}
	return service.ProjectDir
}

func formatPortForward(pf *client.ServicePortForwardStatus) string {
	if pf == nil {
		return "-"
	}
	parts := []string{fmt.Sprintf("%d", pf.Port)}
	if pf.Provider != "" {
		parts = append(parts, pf.Provider)
	}
	if pf.PublicURL != "" {
		parts = append(parts, pf.PublicURL)
	} else if pf.Error != "" {
		parts = append(parts, "error="+pf.Error)
	} else if pf.Status != "" {
		parts = append(parts, "status="+pf.Status)
	}
	return strings.Join(parts, "  ")
}
