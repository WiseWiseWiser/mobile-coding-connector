package agentcli

import (
	"fmt"
	"strings"

	"github.com/xhd2015/ai-critic/client"
	"github.com/xhd2015/less-gen/flags"
)

const projectHelp = `Usage: remote-agent project <subcommand> [args...]

Inspect and update projects known to the remote server.

Subcommands:
  list
      List projects and their configured Git commit identity.

  git-config get|check <project-id-or-name-or-dir>
      Show the Git commit identity configured for one project.

  git-config set <project-id-or-name-or-dir> --name NAME --email EMAIL [--identity-id ID]
      Set the Git commit identity used by this project.

  git-config unset <project-id-or-name-or-dir>
      Clear the Git commit identity for this project.
`

const projectListHelp = `Usage: remote-agent project list

List all projects known to the remote server, including each project's
configured Git commit identity.
`

const projectGitConfigHelp = `Usage: remote-agent project git-config <subcommand> [args...]

Check or set the Git commit identity saved on a project.

Subcommands:
  get <project-id-or-name-or-dir>
  check <project-id-or-name-or-dir>
      Show the configured identity.

  set <project-id-or-name-or-dir> --name NAME --email EMAIL [--identity-id ID]
      Save the identity used when committing from this project. The optional
      identity id is the browser-side Git Settings identity id when known.

  unset <project-id-or-name-or-dir>
      Clear the saved identity.
`

const projectGitConfigSetHelp = `Usage: remote-agent project git-config set <project-id-or-name-or-dir> --name NAME --email EMAIL [--identity-id ID]

Set the Git commit identity saved on a project.

Options:
  --name NAME         Git user.name to use for commits.
  --email EMAIL       Git user.email to use for commits.
  --identity-id ID    Optional browser-side Git identity id.
  -h, --help          Show this help message.
`

func runProject(resolve func() (*client.Client, error), args []string) error {
	if len(args) == 0 {
		fmt.Print(projectHelp)
		return nil
	}

	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list":
		return runProjectList(resolve, rest)
	case "git-config", "gitconfig":
		return runProjectGitConfig(resolve, rest)
	case "-h", "--help":
		fmt.Print(projectHelp)
		return nil
	default:
		return fmt.Errorf("unknown project subcommand: %s", sub)
	}
}

func runProjectList(resolve func() (*client.Client, error), args []string) error {
	if len(args) > 0 {
		if args[0] == "-h" || args[0] == "--help" {
			fmt.Print(projectListHelp)
			return nil
		}
		return fmt.Errorf("project list takes no arguments, got %v", args)
	}

	cli, err := resolve()
	if err != nil {
		return err
	}
	projects, err := cli.ListProjects()
	if err != nil {
		return err
	}
	if len(projects) == 0 {
		fmt.Println("No projects found.")
		return nil
	}

	for i, project := range projects {
		if i > 0 {
			fmt.Println()
		}
		printProjectGitConfig(project)
	}
	return nil
}

func runProjectGitConfig(resolve func() (*client.Client, error), args []string) error {
	if len(args) == 0 {
		fmt.Print(projectGitConfigHelp)
		return nil
	}

	sub := args[0]
	rest := args[1:]
	switch sub {
	case "get", "check":
		return runProjectGitConfigGet(resolve, rest)
	case "set":
		return runProjectGitConfigSet(resolve, rest)
	case "unset", "clear":
		return runProjectGitConfigUnset(resolve, rest)
	case "-h", "--help":
		fmt.Print(projectGitConfigHelp)
		return nil
	default:
		return fmt.Errorf("unknown project git-config subcommand: %s", sub)
	}
}

func runProjectGitConfigGet(resolve func() (*client.Client, error), args []string) error {
	if len(args) != 1 {
		if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
			fmt.Println("Usage: remote-agent project git-config get <project-id-or-name-or-dir>")
			return nil
		}
		return fmt.Errorf("project git-config get requires exactly 1 argument <project-id-or-name-or-dir>")
	}

	cli, err := resolve()
	if err != nil {
		return err
	}
	project, err := resolveProjectTarget(cli, args[0])
	if err != nil {
		return err
	}
	printProjectGitConfig(*project)
	return nil
}

func runProjectGitConfigSet(resolve func() (*client.Client, error), args []string) error {
	var name string
	var email string
	var identityID string

	args, err := flags.
		String("--name", &name).
		String("--email", &email).
		String("--identity-id", &identityID).
		Help("-h,--help", projectGitConfigSetHelp).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) != 1 {
		return fmt.Errorf("project git-config set requires exactly 1 argument <project-id-or-name-or-dir>")
	}
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)
	if name == "" {
		return fmt.Errorf("--name is required")
	}
	if email == "" {
		return fmt.Errorf("--email is required")
	}

	cli, err := resolve()
	if err != nil {
		return err
	}
	project, err := resolveProjectTarget(cli, args[0])
	if err != nil {
		return err
	}
	updated, err := cli.SetProjectGitConfig(project.ID, strings.TrimSpace(identityID), name, email)
	if err != nil {
		return err
	}
	fmt.Printf("Updated Git commit identity for %s (%s)\n", updated.Name, updated.ID)
	printProjectGitConfig(*updated)
	return nil
}

func runProjectGitConfigUnset(resolve func() (*client.Client, error), args []string) error {
	if len(args) != 1 {
		if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
			fmt.Println("Usage: remote-agent project git-config unset <project-id-or-name-or-dir>")
			return nil
		}
		return fmt.Errorf("project git-config unset requires exactly 1 argument <project-id-or-name-or-dir>")
	}

	cli, err := resolve()
	if err != nil {
		return err
	}
	project, err := resolveProjectTarget(cli, args[0])
	if err != nil {
		return err
	}
	updated, err := cli.UnsetProjectGitConfig(project.ID)
	if err != nil {
		return err
	}
	fmt.Printf("Cleared Git commit identity for %s (%s)\n", updated.Name, updated.ID)
	return nil
}

func resolveProjectTarget(cli *client.Client, idNameOrDir string) (*client.ProjectInfo, error) {
	projects, err := cli.ListProjects()
	if err != nil {
		return nil, err
	}
	return matchProjectTarget(projects, idNameOrDir)
}

func matchProjectTarget(projects []client.ProjectInfo, idNameOrDir string) (*client.ProjectInfo, error) {
	idNameOrDir = strings.TrimSpace(idNameOrDir)
	if idNameOrDir == "" {
		return nil, fmt.Errorf("project target cannot be empty")
	}

	for _, project := range projects {
		if project.ID == idNameOrDir {
			project := project
			return &project, nil
		}
	}

	var matches []client.ProjectInfo
	for _, project := range projects {
		if project.Name == idNameOrDir || project.Dir == idNameOrDir {
			matches = append(matches, project)
		}
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no project found for %q", idNameOrDir)
	case 1:
		return &matches[0], nil
	default:
		ids := make([]string, 0, len(matches))
		for _, match := range matches {
			ids = append(ids, match.ID)
		}
		return nil, fmt.Errorf("project target %q is ambiguous; matching IDs: %s", idNameOrDir, strings.Join(ids, ", "))
	}
}

func printProjectGitConfig(project client.ProjectInfo) {
	fmt.Printf("Project: %s (%s)\n", displayOrDash(project.Name), displayOrDash(project.ID))
	fmt.Printf("  Dir:              %s\n", displayOrDash(project.Dir))
	fmt.Printf("  Git Identity ID:  %s\n", displayOrDash(project.GitUserConfigID))
	fmt.Printf("  Git User Name:    %s\n", displayOrDash(project.GitUserName))
	fmt.Printf("  Git User Email:   %s\n", displayOrDash(project.GitUserEmail))
}
