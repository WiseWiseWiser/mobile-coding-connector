package agentcli

import (
	"fmt"
	"strings"

	"github.com/xhd2015/ai-critic/client"
	"github.com/xhd2015/less-gen/flags"
)

func runServiceAdd(resolve func() (*client.Client, error), args []string) error {
	var (
		name           string
		command        string
		projectDir     string
		workingDir     string
		upgradeTarget  string
		envSet         []string
		port           int
		portLabel      string
		portProvider   string
		portBaseDomain string
		portSubdomain  string
		disabled       bool
		start          bool
	)

	args, err := flags.
		String("--name", &name).
		String("--command", &command).
		String("--project-dir", &projectDir).
		String("--working-dir", &workingDir).
		String("--upgrade-target", &upgradeTarget).
		StringSlice("--env", &envSet).
		Int("--port", &port).
		String("--port-label", &portLabel).
		String("--port-provider", &portProvider).
		String("--port-base-domain", &portBaseDomain).
		String("--port-subdomain", &portSubdomain).
		Bool("--disabled", &disabled).
		Bool("--start", &start).
		Help("-h,--help", serviceAddHelp).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) > 0 {
		return fmt.Errorf("service add takes no positional arguments, got %v", args)
	}

	name = strings.TrimSpace(name)
	command = strings.TrimSpace(command)
	if name == "" {
		return fmt.Errorf("--name is required")
	}
	if command == "" {
		return fmt.Errorf("--command is required")
	}

	def := client.ServiceDefinition{
		Name:          name,
		Command:       command,
		ProjectDir:    strings.TrimSpace(projectDir),
		WorkingDir:    strings.TrimSpace(workingDir),
		UpgradeTarget: strings.TrimSpace(upgradeTarget),
	}
	if disabled {
		enabled := false
		def.Enabled = &enabled
	}
	if len(envSet) > 0 {
		def.ExtraEnv = map[string]string{}
		for _, assignment := range envSet {
			key, value, err := parseServiceEnvAssignment(assignment)
			if err != nil {
				return err
			}
			def.ExtraEnv[key] = value
		}
	}

	portFieldsSpecified := port > 0 ||
		strings.TrimSpace(portLabel) != "" ||
		strings.TrimSpace(portProvider) != "" ||
		strings.TrimSpace(portBaseDomain) != "" ||
		strings.TrimSpace(portSubdomain) != ""
	if portFieldsSpecified {
		if port <= 0 || port > 65535 {
			return fmt.Errorf("--port must be between 1 and 65535")
		}
		def.PortForward = &client.ServicePortForward{
			Port:       port,
			Label:      strings.TrimSpace(portLabel),
			Provider:   strings.TrimSpace(portProvider),
			BaseDomain: strings.TrimSpace(portBaseDomain),
			Subdomain:  strings.TrimSpace(portSubdomain),
		}
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	// Create definition only (no restart); new services have no process yet.
	created, err := cli.SaveService(def, false)
	if err != nil {
		return err
	}

	if start {
		started, startErr := cli.StartService(created.ID)
		if startErr != nil {
			return startErr
		}
		created = started
	}

	fmt.Printf("Created service %s (%s)\n", created.ID, displayOrDash(created.Name))
	printService(*created)
	return nil
}

func runServiceRm(resolve func() (*client.Client, error), args []string) error {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Print(serviceRmHelp)
		return nil
	}
	if len(args) != 1 {
		return fmt.Errorf("service rm requires exactly 1 argument <service-name-or-id>")
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	service, err := resolveServiceTarget(cli, args[0])
	if err != nil {
		return err
	}

	if err := cli.DeleteService(service.ID); err != nil {
		return err
	}

	fmt.Printf("Removed service %s (%s)\n", service.ID, displayOrDash(service.Name))
	return nil
}
