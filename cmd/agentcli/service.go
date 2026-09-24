package agentcli

import (
	"fmt"
	"strings"

	"github.com/xhd2015/ai-critic/client"
	"github.com/xhd2015/less-gen/flags"
)

const serviceHelp = `Usage: remote-agent service <subcommand> [args...]

Manage remote services configured on the remote-agent server.

Subcommands:
  list [--all]
      List all managed services. --all is accepted as a no-op alias.

  add --name NAME --command COMMAND [options...]
      Create a new service definition (optionally start or disable it).

  rm <service-name-or-id>
      Remove one service by name or id.

  start <service-name-or-id>
      Start one service.

  stop <service-name-or-id>
      Stop one service.

  restart <service-name-or-id>
      Restart one service.

  disable <service-name-or-id>
      Disable auto-start and daemon management for one service.

  enable <service-name-or-id>
      Enable auto-start and daemon management for one service.

  rename <service-name-or-id> <new-name>
      Rename one service without restarting it.

  update <service-name-or-id> [--field value...]
      Update one service definition without restarting it.

  upgrade <service-name-or-id> <local-binary> [--target <remote-path>]
      Upload a replacement binary, then stop, replace, and start one service.

  logs [--lines N] <service-name-or-id>
      Stream one service's log file.

Note: some system services are status-only and expose no start/stop.
`

const serviceListHelp = `Usage: remote-agent service list [--all]

List all managed services from the remote server.

Options:
  --all               Accepted for compatibility; list is always global.
  -h, --help          Show this help message.
`

const serviceAddHelp = `Usage: remote-agent service add --name NAME --command COMMAND [options...]

Create a new managed service definition on the remote server.

Required:
  --name NAME                 Service name.
  --command COMMAND           Shell command to run.

Options:
  --working-dir DIR           Working directory for the process.
  --upgrade-target PATH       Remembered service upgrade target.
  --upgrade-pre-stop-cmd CMD  Step run by 'service upgrade' while the service is
                              still running, e.g. 'git fetch' or a build.
                              Can be repeated; runs in order.
  --upgrade-post-stop-cmd CMD Step run by 'service upgrade' after the service is
                              stopped, e.g. swapping a freshly built binary.
                              Can be repeated; runs in order.
  --upgrade-timeout DURATION  Per-step timeout, e.g. 10m; 0 disables it.
  --env KEY=VALUE             Environment variable. Can be repeated.
  --port N                    Port-forward port.
  --port-label LABEL          Port-forward label.
  --port-provider PROVIDER    Port-forward provider.
  --port-base-domain DOMAIN   Port-forward base domain.
  --port-subdomain NAME       Port-forward subdomain.
  --require-auth              Gate the public URL with a login page.
  --auth-user NAME            Require this username (omit = any user).
  --auth-token TOKEN          Custom token (omit = shared server token).
  --disabled                  Create with enabled=false (no auto-start).
  --start                     Start the service after creating it.
  -h, --help                  Show this help message.
`

const serviceRmHelp = `Usage: remote-agent service rm <service-name-or-id>

Remove one managed service by name or id.
`

const serviceLogsHelp = `Usage: remote-agent service logs [--lines N] <service-name-or-id>

Stream logs for one managed service using the same backend log stream as
the frontend.

Options:
  --lines N           Initial tail size before following new log lines.
                      Defaults to 100.
  -h, --help          Show this help message.
`

const serviceRenameHelp = `Usage: remote-agent service rename <service-name-or-id> <new-name>

Rename one managed service. The saved definition is updated, but the running
process is not restarted.
`

const serviceUpdateHelp = `Usage: remote-agent service update <service-name-or-id> [options...]

Update one managed service definition. Saved values do not affect the running
process until the service is restarted.

Options:
  --name NAME                 Set service name.
  --command COMMAND           Set shell command.
  --working-dir DIR           Set working directory.
  --upgrade-target PATH       Set remembered service upgrade target.
  --upgrade-pre-stop-cmd CMD  Set a pre-stop upgrade step (repeatable; replaces
                              the stored pre-stop steps).
  --upgrade-post-stop-cmd CMD Set a post-stop upgrade step (repeatable; replaces
                              the stored post-stop steps).
  --upgrade-timeout DURATION  Set the per-step timeout, e.g. 10m; 0 disables it.
  --clear-upgrade-pre-stop-cmds   Remove all pre-stop upgrade steps.
  --clear-upgrade-post-stop-cmds  Remove all post-stop upgrade steps.
  --env KEY=VALUE             Set or replace an environment variable.
                              Can be repeated.
  --unset-env KEY             Remove an environment variable.
                              Can be repeated.
  --clear-env                 Remove all environment variables.
  --port N                    Set port-forward port.
  --port-label LABEL          Set port-forward label.
  --port-provider PROVIDER    Set port-forward provider.
  --port-base-domain DOMAIN   Set port-forward base domain.
  --port-subdomain NAME       Set port-forward subdomain.
  --clear-port-forward        Remove port-forward configuration.
  --require-auth              Gate the public URL with a login page.
  --auth-user NAME            Require this username (empty = any user).
  --auth-token TOKEN          Custom token (empty = shared server token).
  --clear-auth                Disable auth on the public URL.
  -h, --help                  Show this help message.
`

func runService(resolve func() (*client.Client, error), args []string) error {
	if len(args) == 0 {
		fmt.Print(serviceHelp)
		return nil
	}

	switch args[0] {
	case "list":
		return runServiceList(resolve, args[1:])
	case "add":
		return runServiceAdd(resolve, args[1:])
	case "rm":
		return runServiceRm(resolve, args[1:])
	case "start":
		return runServiceAction(resolve, "start", args[1:])
	case "stop":
		return runServiceAction(resolve, "stop", args[1:])
	case "restart":
		return runServiceAction(resolve, "restart", args[1:])
	case "disable":
		return runServiceEnableDisable(resolve, "disable", args[1:])
	case "enable":
		return runServiceEnableDisable(resolve, "enable", args[1:])
	case "rename":
		return runServiceRename(resolve, args[1:])
	case "update":
		return runServiceUpdate(resolve, args[1:])
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
	var listAll bool
	args, err := flags.
		Bool("--all", &listAll). // no-op alias for compatibility
		Help("-h,--help", serviceListHelp).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) > 0 {
		return fmt.Errorf("service list takes no positional arguments, got %v", args)
	}
	_ = listAll

	cli, err := resolve()
	if err != nil {
		return err
	}

	services, err := cli.ListServices()
	if err != nil {
		return err
	}
	if len(services) == 0 {
		fmt.Println("No services found.")
		return nil
	}

	userServices, systemServices := splitServicesByKind(services)

	printGroup := func(title string, items []client.ServiceStatus) {
		if len(items) == 0 {
			return
		}
		fmt.Println("  " + title)
		for i, service := range items {
			if i > 0 {
				fmt.Println()
			}
			printService(service)
		}
	}

	printGroup("USER SERVICES", userServices)
	if len(userServices) > 0 && len(systemServices) > 0 {
		fmt.Println()
	}
	printGroup("SYSTEM SERVICES", systemServices)
	return nil
}

// splitServicesByKind separates user-defined services from server-owned ones,
// preserving the order the server returned.
func splitServicesByKind(services []client.ServiceStatus) (userServices, systemServices []client.ServiceStatus) {
	userServices = make([]client.ServiceStatus, 0, len(services))
	systemServices = make([]client.ServiceStatus, 0, len(services))
	for _, service := range services {
		if isSystemService(service) {
			systemServices = append(systemServices, service)
			continue
		}
		userServices = append(userServices, service)
	}
	return userServices, systemServices
}

// isSystemService reports whether the status describes a server-owned
// in-process service rather than a user-defined command.
func isSystemService(service client.ServiceStatus) bool {
	return service.Kind == "system"
}

// serviceKindWord normalizes the kind for display; an absent kind is a user
// service.
func serviceKindWord(kind string) string {
	if kind == "system" {
		return "system"
	}
	return "user"
}

func runServiceEnableDisable(resolve func() (*client.Client, error), action string, args []string) error {
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
	case "disable":
		result, err := cli.DisableService(service.ID)
		if err != nil {
			return err
		}
		fmt.Print(result.Message)
	case "enable":
		result, err := cli.EnableService(service.ID)
		if err != nil {
			return err
		}
		fmt.Print(result.Message)
	default:
		return fmt.Errorf("unsupported service action: %s", action)
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
	services, err := cli.ListServices()
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
	fmt.Printf("%s %s\n", label("Kind"), serviceKindWord(service.Kind))
	if service.Description != "" {
		fmt.Printf("%s %s\n", label("About"), service.Description)
	}
	fmt.Printf("%s %s\n", label("Status"), displayOrDash(service.Status))

	// System services are in-process subsystems: they have no command, no PID
	// and no work dir, but they may expose a public URL and a log.
	if isSystemService(service) {
		if service.Detail != "" {
			fmt.Printf("%s %s\n", label("Detail"), service.Detail)
		}
		if service.Edge != "" {
			fmt.Printf("%s %s\n", label("Edge"), service.Edge)
		}
		if service.PublicURL != "" {
			fmt.Printf("%s %s\n", label("Public"), service.PublicURL)
		}
		for _, host := range service.Hosts {
			fmt.Printf("%s %s  %d dials  %s\n", label("Host"), host.Host, host.Dials, host.State)
		}
		if service.Mocked {
			fmt.Printf("%s %s\n", label("Note"), "simulated; not a live integration")
		}
		if service.AutoStartSwitch {
			fmt.Printf("%s %s\n", label("Auto-start"), boolWord(service.Enabled))
		}
		if service.LogPath != "" {
			fmt.Printf("%s %s\n", label("Log Path"), displayOrDash(service.LogPath))
		}
		return
	}

	fmt.Printf("%s %s\n", label("PID"), formatOptionalInt(service.PID))
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
	for _, line := range formatServiceAuthLines(service) {
		fmt.Println(line)
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

func formatServiceAuthLines(service client.ServiceStatus) []string {
	if !service.RequireAuth {
		return nil
	}
	const labelWidth = 12
	label := func(name string) string {
		return fmt.Sprintf("  %-*s", labelWidth, name+":")
	}
	user := strings.TrimSpace(service.AuthUser)
	if user == "" {
		user = "any"
	}
	mode := strings.TrimSpace(service.AuthTokenMode)
	if mode == "" {
		mode = "shared"
	}
	tokens := append([]string(nil), service.AuthTokens...)
	if len(tokens) == 0 && strings.TrimSpace(service.AuthToken) != "" {
		tokens = []string{service.AuthToken}
	}
	lines := []string{
		fmt.Sprintf("%s %s", label("Auth"), "required"),
		fmt.Sprintf("%s %s", label("Auth User"), user),
	}
	if len(tokens) == 0 {
		lines = append(lines, fmt.Sprintf("%s %s  (none)", label("Auth Token"), mode))
		return lines
	}
	for _, token := range tokens {
		lines = append(lines, fmt.Sprintf("%s %s  %s", label("Auth Token"), mode, token))
	}
	return lines
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
