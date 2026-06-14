package main

import (
	"fmt"
	"strings"

	"github.com/xhd2015/ai-critic/client"
	"github.com/xhd2015/less-gen/flags"
)

const settingsHelp = `Usage: remote-agent settings <subcommand> [args...]

Manage settings stored by the remote server.

Subcommands:
  git-users <subcommand> [args...]
      Manage Git identities used for commits.
`

const settingsGitUsersHelp = `Usage: remote-agent settings git-users <subcommand> [args...]

Manage the Git names and emails available for commits.

Subcommands:
  list
      List configured Git identities.

  add --name NAME --email EMAIL [--id ID]
      Add a Git identity. If --id is omitted, the server creates one.

  set <id> --name NAME --email EMAIL
      Update an existing Git identity.

  delete <id>
      Delete a Git identity.
`

const settingsGitUsersAddHelp = `Usage: remote-agent settings git-users add --name NAME --email EMAIL [--id ID]

Add a Git identity to the remote server settings.

Options:
  --name NAME     Git user.name to use for commits.
  --email EMAIL   Git user.email to use for commits.
  --id ID         Optional stable identity id.
  -h, --help      Show this help message.
`

const settingsGitUsersSetHelp = `Usage: remote-agent settings git-users set <id> --name NAME --email EMAIL

Update a Git identity in the remote server settings.

Options:
  --name NAME     Git user.name to use for commits.
  --email EMAIL   Git user.email to use for commits.
  -h, --help      Show this help message.
`

func runSettings(resolve func() (*client.Client, error), args []string) error {
	if len(args) == 0 {
		fmt.Print(settingsHelp)
		return nil
	}

	sub := args[0]
	rest := args[1:]
	switch sub {
	case "git-users", "git-user", "git-config", "git-configs":
		return runSettingsGitUsers(resolve, rest)
	case "-h", "--help":
		fmt.Print(settingsHelp)
		return nil
	default:
		return fmt.Errorf("unknown settings subcommand: %s", sub)
	}
}

func runSettingsGitUsers(resolve func() (*client.Client, error), args []string) error {
	if len(args) == 0 {
		fmt.Print(settingsGitUsersHelp)
		return nil
	}

	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list", "ls":
		return runSettingsGitUsersList(resolve, rest)
	case "add":
		return runSettingsGitUsersAdd(resolve, rest)
	case "set", "update":
		return runSettingsGitUsersSet(resolve, rest)
	case "delete", "del", "rm", "remove":
		return runSettingsGitUsersDelete(resolve, rest)
	case "-h", "--help":
		fmt.Print(settingsGitUsersHelp)
		return nil
	default:
		return fmt.Errorf("unknown settings git-users subcommand: %s", sub)
	}
}

func runSettingsGitUsersList(resolve func() (*client.Client, error), args []string) error {
	if len(args) > 0 {
		if args[0] == "-h" || args[0] == "--help" {
			fmt.Println("Usage: remote-agent settings git-users list")
			return nil
		}
		return fmt.Errorf("settings git-users list takes no arguments, got %v", args)
	}

	cli, err := resolve()
	if err != nil {
		return err
	}
	configs, err := cli.ListGitUserConfigs()
	if err != nil {
		return err
	}
	if len(configs) == 0 {
		fmt.Println("No Git identities configured.")
		return nil
	}
	for i, config := range configs {
		if i > 0 {
			fmt.Println()
		}
		printGitUserConfig(config)
	}
	return nil
}

func runSettingsGitUsersAdd(resolve func() (*client.Client, error), args []string) error {
	var id string
	var name string
	var email string

	args, err := flags.
		String("--id", &id).
		String("--name", &name).
		String("--email", &email).
		Help("-h,--help", settingsGitUsersAddHelp).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) > 0 {
		return fmt.Errorf("settings git-users add does not accept positional args: %v", args)
	}

	cli, err := resolve()
	if err != nil {
		return err
	}
	config, err := cli.AddGitUserConfig(id, name, email)
	if err != nil {
		return err
	}
	fmt.Println("Added Git identity:")
	printGitUserConfig(*config)
	return nil
}

func runSettingsGitUsersSet(resolve func() (*client.Client, error), args []string) error {
	var name string
	var email string

	args, err := flags.
		String("--name", &name).
		String("--email", &email).
		Help("-h,--help", settingsGitUsersSetHelp).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) != 1 {
		return fmt.Errorf("settings git-users set requires exactly 1 argument <id>")
	}

	cli, err := resolve()
	if err != nil {
		return err
	}
	config, err := cli.UpdateGitUserConfig(args[0], name, email)
	if err != nil {
		return err
	}
	fmt.Println("Updated Git identity:")
	printGitUserConfig(*config)
	return nil
}

func runSettingsGitUsersDelete(resolve func() (*client.Client, error), args []string) error {
	if len(args) != 1 {
		if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
			fmt.Println("Usage: remote-agent settings git-users delete <id>")
			return nil
		}
		return fmt.Errorf("settings git-users delete requires exactly 1 argument <id>")
	}

	cli, err := resolve()
	if err != nil {
		return err
	}
	id := strings.TrimSpace(args[0])
	if err := cli.DeleteGitUserConfig(id); err != nil {
		return err
	}
	fmt.Printf("Deleted Git identity: %s\n", id)
	return nil
}

func printGitUserConfig(config client.GitUserConfig) {
	fmt.Printf("ID:      %s\n", displayOrDash(config.ID))
	fmt.Printf("Name:    %s\n", displayOrDash(config.Name))
	fmt.Printf("Email:   %s\n", displayOrDash(config.Email))
	if config.CreatedAt != "" {
		fmt.Printf("Created: %s\n", config.CreatedAt)
	}
}
