package agentcli

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/xhd2015/ai-critic/client"
	"github.com/xhd2015/less-gen/flags"
)

const serviceUpgradeHelp = `Usage: remote-agent service upgrade <service-name-or-id> [<local-binary>] [options]

Run the service's upgrade pipeline: the pre-stop steps run while the service is
still serving, then the service stops, an uploaded binary (if any) is moved
into place, the post-stop steps run, and the service starts again.

Steps come from the service definition (--upgrade-pre-stop-cmd /
--upgrade-post-stop-cmd on service add/update) and can be overridden per run.
Each step runs through ` + "`bash -lc 'set -eo pipefail; <cmd>'`" + ` in the service
working directory and environment.

Failure behavior:
  pre-stop step   aborts the upgrade and leaves the service untouched.
  post-stop step  aborts the upgrade, restarts the service, then reports.

The resolved upgrade target is exported to every step as
$` + serviceUpgradeTargetEnv + `, so a pre-stop step can build to a temporary
path and a post-stop step can swap it into place (a running binary cannot be
overwritten in place).

Arguments:
  <service-name-or-id>   Service to upgrade (required).
  <local-binary>         Optional binary to upload and move into place.

Options:
  --target PATH          Remote binary path to replace. Relative paths and ~/...
                         are resolved under the remote server's home directory.
                         Remembered for later upgrades of the same service.
  --upgrade-pre-stop-cmd CMD   Step to run while the service is still running.
                               Repeatable; overrides the stored steps.
  --upgrade-post-stop-cmd CMD  Step to run after the service is stopped.
                               Repeatable; overrides the stored steps.
  --upgrade-timeout DURATION   Per-step timeout, e.g. 10m; 0 disables it.
                               Defaults to the stored value (15m).
  -h, --help             Show this help message.
`

// serviceUpgradeTargetEnv mirrors services.EnvUpgradeTarget for help text and
// documentation without importing the server package.
const serviceUpgradeTargetEnv = "REMOTE_AGENT_UPGRADE_TARGET"

func runServiceUpgrade(resolve func() (*client.Client, error), args []string) error {
	var targetFlag string
	var preStopCmds []string
	var postStopCmds []string
	var timeoutFlag string
	originalArgs := args

	args, err := flags.
		String("--target", &targetFlag).
		StringSlice("--upgrade-pre-stop-cmd", &preStopCmds).
		StringSlice("--upgrade-post-stop-cmd", &postStopCmds).
		String("--upgrade-timeout", &timeoutFlag).
		Help("-h,--help", serviceUpgradeHelp).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		return fmt.Errorf("service upgrade requires <service-name-or-id>; see 'service upgrade --help'")
	}
	if len(args) > 2 {
		return fmt.Errorf("service upgrade takes <service-name-or-id> and an optional <local-binary>, got %d arguments", len(args))
	}

	serviceName := args[0]
	localBinary := ""
	if len(args) == 2 {
		localBinary = strings.TrimSpace(args[1])
	}

	timeoutSeconds, timeoutSpecified, err := parseUpgradeTimeout(timeoutFlag, originalArgs)
	if err != nil {
		return err
	}

	var stat os.FileInfo
	if localBinary != "" {
		stat, err = os.Stat(localBinary)
		if err != nil {
			return fmt.Errorf("failed to stat local binary: %w", err)
		}
		if stat.IsDir() {
			return fmt.Errorf("local binary is a directory, not a file: %s", localBinary)
		}
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	service, err := resolveServiceTarget(cli, serviceName)
	if err != nil {
		return err
	}

	request := client.ServiceUpgradeRequest{
		ID:           service.ID,
		PreStopCmds:  preStopCmds,
		PostStopCmds: postStopCmds,
	}
	if timeoutSpecified {
		request.TimeoutSeconds = &timeoutSeconds
	}

	targetInput := strings.TrimSpace(targetFlag)
	if localBinary != "" {
		localBase := filepath.Base(localBinary)
		tmpPath := remoteUpgradeTempPath(localBase)

		if targetInput != "" {
			fmt.Printf("Remote service will remember upgrade target for %s (%s): %s\n", displayOrDash(service.Name), service.ID, targetInput)
		}
		fmt.Printf("Temporary upload path: %s\n", tmpPath)
		fmt.Printf("Uploading %s (%s) -> %s\n", localBinary, formatSize(stat.Size()), tmpPath)

		result, err := cli.UploadFile(localBinary, tmpPath, client.UploadOptions{
			ChmodExec: true,
		}, printUploadProgress)
		if err != nil {
			return err
		}
		fmt.Printf("Upload complete: %s (%s)\n", result.Path, formatSize(result.Size))

		request.TmpPath = tmpPath
		request.LocalBase = localBase
		request.Target = targetInput
	} else if len(preStopCmds) == 0 && len(postStopCmds) == 0 && !timeoutSpecified {
		// Without a binary the definition must carry the steps; the server
		// rejects the request when nothing is configured, but naming the
		// service here produces a clearer message.
		if len(service.UpgradePreStopCmds) == 0 && len(service.UpgradePostStopCmds) == 0 {
			return fmt.Errorf("service %s has no upgrade steps configured and no binary was given; set them with 'service update --upgrade-pre-stop-cmd/--upgrade-post-stop-cmd'", displayOrDash(service.Name))
		}
	}

	return streamServiceUpgrade(cli, service, request)
}

// streamServiceUpgrade runs the upgrade over SSE and prints sections, step
// output and the server-rendered summary as they arrive.
func streamServiceUpgrade(cli *client.Client, service *client.ServiceStatus, request client.ServiceUpgradeRequest) error {
	fmt.Printf("Upgrading %s (%s)\n", displayOrDash(service.Name), service.ID)
	_, err := cli.UpgradeServiceStream(request, func(event client.ServerStreamEvent) {
		switch event.Type {
		case "section":
			if event.Message != "" {
				fmt.Printf("  ── %s\n", event.Message)
			}
		case "log":
			fmt.Println(event.Message)
		case "progress":
			if event.Message != "" {
				fmt.Printf("  %s\n", event.Message)
			}
		}
	})
	return err
}

// parseUpgradeTimeout converts a duration flag into whole seconds, reporting
// whether the flag was supplied at all (so the stored value can win otherwise).
func parseUpgradeTimeout(value string, originalArgs []string) (int, bool, error) {
	trimmed := strings.TrimSpace(value)
	specified := trimmed != "" || flagPresent(originalArgs, "--upgrade-timeout")
	if !specified {
		return 0, false, nil
	}
	if trimmed == "" {
		return 0, true, fmt.Errorf("--upgrade-timeout requires a value, e.g. 10m or 0 to disable")
	}
	if trimmed == "0" {
		return 0, true, nil
	}
	duration, err := time.ParseDuration(trimmed)
	if err != nil {
		return 0, true, fmt.Errorf("invalid --upgrade-timeout %q: %w", trimmed, err)
	}
	if duration < 0 {
		return 0, true, fmt.Errorf("--upgrade-timeout must not be negative")
	}
	seconds := int((duration + time.Second - 1) / time.Second)
	if seconds == 0 {
		seconds = 1
	}
	return seconds, true, nil
}

// flagPresent reports whether an exact flag name appears in the raw arguments.
func flagPresent(args []string, name string) bool {
	for _, arg := range args {
		if arg == name || strings.HasPrefix(arg, name+"=") {
			return true
		}
	}
	return false
}

func remoteUpgradeTempPath(localBase string) string {
	base := sanitizeRemoteTempBase(path.Base(localBase))
	if base == "" || base == "." || base == "/" {
		base = "binary"
	}
	return path.Join("/tmp", fmt.Sprintf("remote-agent-upgrade-%d-%d-%s", time.Now().UnixNano(), os.Getpid(), base))
}

func sanitizeRemoteTempBase(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return strings.Trim(b.String(), ".")
}
