package sshcmd

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	configLocalBegin = "# >>> remote-agent ssh config-local >>>"
	configLocalEnd   = "# <<< remote-agent ssh config-local <<<"
)

// ConfigLocalUsage documents local OpenSSH profile management.
const ConfigLocalUsage = `Usage: remote-agent ssh config-local <command> [options]

Commands:
  install     install a managed local OpenSSH profile
  uninstall   remove the managed local OpenSSH profile
  status      show profile and relay state

Run 'remote-agent ssh config-local <command> --help' for command options.
`

const configLocalInstallUsage = `Usage: remote-agent ssh config-local install [options]

Options:
  --user USER    SSH user; must match tunnel user (default: agent)
  --host HOST    local SSH alias (default: remote-agent)
  --dry-run      print planned changes without writing files
  -h, --help     show this help
`

const configLocalUninstallUsage = `Usage: remote-agent ssh config-local uninstall [--dry-run]

Options:
  --dry-run      print planned changes without writing files
  -h, --help     show this help
`

const configLocalStatusUsage = `Usage: remote-agent ssh config-local status

Shows the persistent OpenSSH profile and the current local relay state.
`

// ConfigLocalProfile is the persistent alias configuration consumed by --serve.
type ConfigLocalProfile struct {
	Host string `json:"host"`
	User string `json:"user"`
}

var aliasRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

func configLocalProfilePath(configDir string) string {
	return filepath.Join(configDir, "config-local.json")
}

// LoadConfigLocalProfile returns defaults when no local profile is installed.
func LoadConfigLocalProfile(configDir string) (ConfigLocalProfile, error) {
	profile := ConfigLocalProfile{Host: "remote-agent", User: "agent"}
	data, err := os.ReadFile(configLocalProfilePath(configDir))
	if os.IsNotExist(err) {
		return profile, nil
	}
	if err != nil {
		return profile, err
	}
	if err := json.Unmarshal(data, &profile); err != nil {
		return profile, err
	}
	if err := validateConfigLocal(profile); err != nil {
		return profile, err
	}
	return profile, nil
}

// RunConfigLocal manages the normal-OpenSSH profile. Status is always read-only.
func RunConfigLocal(args []string, home, configDir string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		_, err := io.WriteString(stdout, ConfigLocalUsage)
		return err
	}
	switch args[0] {
	case "install":
		return runConfigLocalInstall(args[1:], home, configDir, stdout)
	case "uninstall":
		return runConfigLocalUninstall(args[1:], home, configDir, stdout)
	case "status":
		return runConfigLocalStatus(args[1:], home, configDir, stdout, stderr)
	default:
		return fmt.Errorf("unknown config-local command %q (want install, uninstall, or status)", args[0])
	}
}

func runConfigLocalInstall(args []string, home, configDir string, stdout io.Writer) error {
	fs := flag.NewFlagSet("config-local install", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	user := fs.String("user", "agent", "")
	host := fs.String("host", "remote-agent", "")
	dryRun := fs.Bool("dry-run", false, "")
	help := fs.Bool("help", false, "")
	fs.BoolVar(help, "h", false, "")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		_, err := io.WriteString(stdout, configLocalInstallUsage)
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("config-local install accepts no positional arguments")
	}
	profile := ConfigLocalProfile{Host: *host, User: *user}
	if err := validateConfigLocal(profile); err != nil {
		return err
	}
	configPath := filepath.Join(home, ".ssh", "config")
	if *dryRun {
		_, err := fmt.Fprintf(stdout, "dry-run: would install local SSH profile\n  host: %s\n  user: %s\n  config: %s\n", profile.Host, profile.User, configPath)
		return err
	}
	if err := ensureHostAvailable(configPath, profile.Host); err != nil {
		return err
	}
	if _, err := EnsureClientKeyPair(configDir); err != nil {
		return err
	}
	if err := writeSSHConfig(configDir, profile.Host, profile.User, filepath.Join(configDir, "relay.sock")); err != nil {
		return err
	}
	if err := saveConfigLocalProfile(configDir, profile); err != nil {
		return err
	}
	if err := upsertConfigLocalBlock(configPath, configLocalBlock(configDir), profile.Host); err != nil {
		return err
	}
	_, err := fmt.Fprintf(stdout, "installed local SSH profile\n  host: %s\n  user: %s\n  config: %s\n", profile.Host, profile.User, configPath)
	return err
}

func runConfigLocalUninstall(args []string, home, configDir string, stdout io.Writer) error {
	fs := flag.NewFlagSet("config-local uninstall", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dryRun := fs.Bool("dry-run", false, "")
	help := fs.Bool("help", false, "")
	fs.BoolVar(help, "h", false, "")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		_, err := io.WriteString(stdout, configLocalUninstallUsage)
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("config-local uninstall accepts no positional arguments")
	}
	configPath := filepath.Join(home, ".ssh", "config")
	if *dryRun {
		_, err := fmt.Fprintf(stdout, "dry-run: would uninstall local SSH profile\n  config: %s\n", configPath)
		return err
	}
	if err := removeConfigLocalBlock(configPath); err != nil {
		return err
	}
	if err := os.Remove(configLocalProfilePath(configDir)); err != nil && !os.IsNotExist(err) {
		return err
	}
	_, err := fmt.Fprintf(stdout, "uninstalled local SSH profile\n  config: %s\n", configPath)
	return err
}

func runConfigLocalStatus(args []string, home, configDir string, stdout, stderr io.Writer) error {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		_, err := io.WriteString(stdout, configLocalStatusUsage)
		return err
	}
	if len(args) != 0 {
		return errors.New("config-local status accepts no arguments")
	}
	configPath := filepath.Join(home, ".ssh", "config")
	installed, err := configLocalBlockPresent(configPath)
	if err != nil {
		return err
	}
	_, profileErr := os.Stat(configLocalProfilePath(configDir))
	profilePresent := profileErr == nil
	if profileErr != nil && !os.IsNotExist(profileErr) {
		return profileErr
	}
	profile, err := LoadConfigLocalProfile(configDir)
	if err != nil {
		return err
	}
	socketPath := filepath.Join(configDir, "relay.sock")
	socketPresent := false
	if info, err := os.Lstat(socketPath); err == nil {
		if info.Mode()&os.ModeSocket == 0 {
			return fmt.Errorf("relay path is not a Unix socket: %s", socketPath)
		}
		socketPresent = true
	} else if !os.IsNotExist(err) {
		return err
	}
	store := &FileSessionStore{Root: filepath.Dir(configDir)}
	sess, err := store.Load("default")
	if err != nil {
		return err
	}
	active := sess != nil && sess.Alive && sess.LocalSocket == socketPath && socketPresent

	_, _ = fmt.Fprintln(stdout, "Local SSH profile")
	_, _ = fmt.Fprintf(stdout, "  installed:  %s\n", yesNo(installed && profilePresent))
	if profilePresent {
		_, _ = fmt.Fprintf(stdout, "  host:       %s\n  user:       %s\n  config:     %s\n  include:    %s\n", profile.Host, profile.User, configPath, filepath.Join(configDir, "ssh_config"))
	}
	_, _ = fmt.Fprintln(stdout, "\nRelay")
	_, _ = fmt.Fprintf(stdout, "  socket:     %s\n  session:    %s\n  reachable:  %s\n", socketPath, sessionState(active), yesNo(active))

	if profilePresent != installed {
		_, _ = fmt.Fprintln(stderr, "warning: local SSH profile state and ~/.ssh/config managed block disagree")
	}
	if (installed || profilePresent) && !active {
		_, _ = fmt.Fprintln(stderr, "warning: start the relay with: remote-agent ssh --serve")
	}
	return nil
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func sessionState(active bool) string {
	if active {
		return "active"
	}
	return "inactive"
}

func configLocalBlockPresent(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	content := string(data)
	start := strings.Count(content, configLocalBegin)
	end := strings.Count(content, configLocalEnd)
	if start == 0 && end == 0 {
		return false, nil
	}
	if start != 1 || end != 1 || strings.Index(content, configLocalBegin) > strings.Index(content, configLocalEnd) {
		return false, errors.New("malformed remote-agent config-local block in ~/.ssh/config")
	}
	return true, nil
}

func validateConfigLocal(profile ConfigLocalProfile) error {
	if profile.User != "agent" {
		return fmt.Errorf("--user must be %q; the remote SSH tunnel rejects other users", "agent")
	}
	if !aliasRE.MatchString(profile.Host) {
		return fmt.Errorf("invalid --host %q: use a plain SSH alias without patterns or whitespace", profile.Host)
	}
	return nil
}

func saveConfigLocalProfile(configDir string, profile ConfigLocalProfile) error {
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configLocalProfilePath(configDir), append(data, '\n'), 0o600)
}

func configLocalBlock(configDir string) string {
	return fmt.Sprintf("%s\nInclude %s\n%s\n", configLocalBegin, shellQuote(filepath.Join(configDir, "ssh_config")), configLocalEnd)
}

func ensureHostAvailable(path, host string) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	stripped, err := stripConfigLocalBlock(string(data))
	if err != nil {
		return err
	}
	if hasHostAlias(stripped, host) {
		return fmt.Errorf("cannot install host %q: ~/.ssh/config already defines a non-managed Host entry", host)
	}
	return nil
}

func upsertConfigLocalBlock(path, block, host string) error {
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	stripped, err := stripConfigLocalBlock(string(data))
	if err != nil {
		return err
	}
	if hasHostAlias(stripped, host) {
		return fmt.Errorf("cannot install host %q: ~/.ssh/config already defines a non-managed Host entry", host)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	content := block
	if strings.TrimSpace(stripped) != "" {
		content += "\n" + strings.TrimLeft(stripped, "\n")
	}
	return os.WriteFile(path, []byte(content), 0o600)
}

func hasHostAlias(content, host string) bool {
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || !strings.EqualFold(fields[0], "Host") {
			continue
		}
		for _, alias := range fields[1:] {
			if alias == host {
				return true
			}
		}
	}
	return false
}

func removeConfigLocalBlock(path string) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	stripped, err := stripConfigLocalBlock(string(data))
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(stripped), 0o600)
}

func stripConfigLocalBlock(content string) (string, error) {
	start := strings.Index(content, configLocalBegin)
	end := strings.Index(content, configLocalEnd)
	if start < 0 && end < 0 {
		return content, nil
	}
	if start < 0 || end < start {
		return "", errors.New("malformed remote-agent config-local block in ~/.ssh/config")
	}
	end += len(configLocalEnd)
	if end < len(content) && content[end] == '\n' {
		end++
	}
	return content[:start] + content[end:], nil
}
