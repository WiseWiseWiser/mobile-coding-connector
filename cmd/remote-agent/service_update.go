package main

import (
	"fmt"
	"strings"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/lifelog-private/ai-critic/client"
)

func runServiceRename(resolve func() (*client.Client, error), args []string) error {
	if len(args) != 2 {
		if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
			fmt.Print(serviceRenameHelp)
			return nil
		}
		return fmt.Errorf("service rename requires exactly 2 arguments <service-name-or-id> <new-name>")
	}
	newName := strings.TrimSpace(args[1])
	if newName == "" {
		return fmt.Errorf("new service name cannot be empty")
	}

	cli, err := resolve()
	if err != nil {
		return err
	}
	service, err := resolveServiceTarget(cli, args[0])
	if err != nil {
		return err
	}

	def := serviceDefinitionFromStatus(service)
	oldName := def.Name
	def.Name = newName
	updated, err := cli.SaveService(def, false)
	if err != nil {
		return err
	}
	fmt.Printf("Renamed service %s from %s to %s\n", updated.ID, displayOrDash(oldName), displayOrDash(updated.Name))
	fmt.Println("Saved definition. Restart the service for changed runtime values to take effect.")
	return nil
}

func runServiceUpdate(resolve func() (*client.Client, error), args []string) error {
	originalArgs := append([]string(nil), args...)
	var (
		name             string
		command          string
		projectDir       string
		workingDir       string
		upgradeTarget    string
		envSet           []string
		envUnset         []string
		clearEnv         bool
		port             int
		portLabel        string
		portProvider     string
		portBaseDomain   string
		portSubdomain    string
		clearPortForward bool
	)

	args, err := flags.
		String("--name", &name).
		String("--command", &command).
		String("--project-dir", &projectDir).
		String("--working-dir", &workingDir).
		String("--upgrade-target", &upgradeTarget).
		StringSlice("--env", &envSet).
		StringSlice("--unset-env", &envUnset).
		Bool("--clear-env", &clearEnv).
		Int("--port", &port).
		String("--port-label", &portLabel).
		String("--port-provider", &portProvider).
		String("--port-base-domain", &portBaseDomain).
		String("--port-subdomain", &portSubdomain).
		Bool("--clear-port-forward", &clearPortForward).
		Help("-h,--help", serviceUpdateHelp).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) != 1 {
		return fmt.Errorf("service update requires exactly 1 argument <service-name-or-id>")
	}

	specified := serviceUpdateSpecifiedFlags(originalArgs)
	if len(specified) == 0 {
		return fmt.Errorf("service update requires at least one update flag")
	}

	cli, err := resolve()
	if err != nil {
		return err
	}
	service, err := resolveServiceTarget(cli, args[0])
	if err != nil {
		return err
	}
	def := serviceDefinitionFromStatus(service)

	updateCount := 0
	if specified["--name"] {
		def.Name = strings.TrimSpace(name)
		updateCount++
	}
	if specified["--command"] {
		def.Command = strings.TrimSpace(command)
		updateCount++
	}
	if specified["--project-dir"] {
		def.ProjectDir = strings.TrimSpace(projectDir)
		updateCount++
	}
	if specified["--working-dir"] {
		def.WorkingDir = strings.TrimSpace(workingDir)
		updateCount++
	}
	if specified["--upgrade-target"] {
		def.UpgradeTarget = strings.TrimSpace(upgradeTarget)
		updateCount++
	}

	if clearEnv {
		def.ExtraEnv = nil
		updateCount++
	}
	if len(envSet) > 0 {
		if def.ExtraEnv == nil {
			def.ExtraEnv = map[string]string{}
		}
		for _, assignment := range envSet {
			key, value, err := parseServiceEnvAssignment(assignment)
			if err != nil {
				return err
			}
			def.ExtraEnv[key] = value
		}
		updateCount++
	}
	if len(envUnset) > 0 {
		for _, key := range envUnset {
			key = strings.TrimSpace(key)
			if key == "" {
				return fmt.Errorf("--unset-env requires a non-empty key")
			}
			delete(def.ExtraEnv, key)
		}
		if len(def.ExtraEnv) == 0 {
			def.ExtraEnv = nil
		}
		updateCount++
	}

	portFieldsSpecified := specified["--port"] ||
		specified["--port-label"] ||
		specified["--port-provider"] ||
		specified["--port-base-domain"] ||
		specified["--port-subdomain"]
	if clearPortForward && portFieldsSpecified {
		return fmt.Errorf("--clear-port-forward cannot be combined with --port or --port-* flags")
	}
	if clearPortForward {
		def.PortForward = nil
		updateCount++
	} else if portFieldsSpecified {
		pf := def.PortForward
		if pf == nil {
			pf = &client.ServicePortForward{}
		}
		if specified["--port"] {
			if port <= 0 || port > 65535 {
				return fmt.Errorf("--port must be between 1 and 65535")
			}
			pf.Port = port
		}
		if specified["--port-label"] {
			pf.Label = strings.TrimSpace(portLabel)
		}
		if specified["--port-provider"] {
			pf.Provider = strings.TrimSpace(portProvider)
		}
		if specified["--port-base-domain"] {
			pf.BaseDomain = strings.TrimSpace(portBaseDomain)
		}
		if specified["--port-subdomain"] {
			pf.Subdomain = strings.TrimSpace(portSubdomain)
		}
		if pf.Port <= 0 {
			return fmt.Errorf("port-forward updates require an existing port or --port")
		}
		def.PortForward = pf
		updateCount++
	}

	if updateCount == 0 {
		return fmt.Errorf("service update requires at least one update flag")
	}

	updated, err := cli.SaveService(def, false)
	if err != nil {
		return err
	}
	fmt.Printf("Updated service %s (%s)\n", updated.ID, displayOrDash(updated.Name))
	fmt.Println("Saved definition. Restart the service for changed runtime values to take effect.")
	return nil
}

func serviceDefinitionFromStatus(service *client.ServiceStatus) client.ServiceDefinition {
	if service == nil {
		return client.ServiceDefinition{}
	}
	return client.ServiceDefinition{
		ID:            service.ID,
		Name:          service.Name,
		Command:       service.Command,
		ProjectDir:    service.ProjectDir,
		WorkingDir:    service.WorkingDir,
		ExtraEnv:      cloneServiceEnv(service.ExtraEnv),
		PortForward:   servicePortForwardFromStatus(service.PortForward),
		UpgradeTarget: service.UpgradeTarget,
	}
}

func servicePortForwardFromStatus(pf *client.ServicePortForwardStatus) *client.ServicePortForward {
	if pf == nil {
		return nil
	}
	return &client.ServicePortForward{
		Port:       pf.Port,
		Label:      pf.Label,
		Provider:   pf.Provider,
		BaseDomain: pf.BaseDomain,
		Subdomain:  pf.Subdomain,
	}
}

func cloneServiceEnv(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func parseServiceEnvAssignment(assignment string) (string, string, error) {
	idx := strings.Index(assignment, "=")
	if idx <= 0 {
		return "", "", fmt.Errorf("--env requires KEY=VALUE, got %q", assignment)
	}
	key := strings.TrimSpace(assignment[:idx])
	if key == "" {
		return "", "", fmt.Errorf("--env requires a non-empty key")
	}
	return key, assignment[idx+1:], nil
}

func serviceUpdateSpecifiedFlags(args []string) map[string]bool {
	valueFlags := map[string]bool{
		"--name":             true,
		"--command":          true,
		"--project-dir":      true,
		"--working-dir":      true,
		"--upgrade-target":   true,
		"--env":              true,
		"--unset-env":        true,
		"--port":             true,
		"--port-label":       true,
		"--port-provider":    true,
		"--port-base-domain": true,
		"--port-subdomain":   true,
	}
	boolFlags := map[string]bool{
		"--clear-env":          true,
		"--clear-port-forward": true,
	}
	specified := map[string]bool{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			break
		}
		if !strings.HasPrefix(arg, "-") {
			continue
		}
		name := arg
		if idx := strings.Index(arg, "="); idx != -1 {
			name = arg[:idx]
		}
		if valueFlags[name] {
			specified[name] = true
			if !strings.Contains(arg, "=") && i+1 < len(args) {
				i++
			}
			continue
		}
		if boolFlags[name] {
			specified[name] = true
		}
	}
	return specified
}
