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

const serviceUpgradeHelp = `Usage: remote-agent service upgrade <service-name-or-id> <local-binary> [--target <remote-path>]

Upload <local-binary> to a temporary remote file while the service is still
running. The remote server then resolves the target path, stops the service,
moves the temporary file into place, and starts the service again.

If --target is omitted, the remote target defaults to ~/<local-binary-basename>.
When --target is supplied, the remote service remembers it. Later upgrades for
the same service reuse that target unless a new --target is supplied.

Options:
  --target PATH       Remote binary path to replace. Relative paths and ~/...
                      are resolved under the remote server's home directory.
  -h, --help          Show this help message.
`

func runServiceUpgrade(resolve func() (*client.Client, error), args []string) error {
	var targetFlag string
	args, err := flags.
		String("--target", &targetFlag).
		Help("-h,--help", serviceUpgradeHelp).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) != 2 {
		return fmt.Errorf("service upgrade requires exactly 2 arguments <service-name-or-id> <local-binary>")
	}

	serviceName := args[0]
	localBinary := args[1]
	stat, err := os.Stat(localBinary)
	if err != nil {
		return fmt.Errorf("failed to stat local binary: %w", err)
	}
	if stat.IsDir() {
		return fmt.Errorf("local binary is a directory, not a file: %s", localBinary)
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	service, err := resolveServiceTarget(cli, serviceName)
	if err != nil {
		return err
	}

	localBase := filepath.Base(localBinary)
	targetInput := strings.TrimSpace(targetFlag)
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

	fmt.Printf("Upgrading service %s (%s)\n", service.ID, displayOrDash(service.Name))
	upgrade, err := cli.UpgradeService(client.ServiceUpgradeRequest{
		ID:        service.ID,
		TmpPath:   tmpPath,
		LocalBase: localBase,
		Target:    targetInput,
	})
	if err != nil {
		return err
	}
	fmt.Printf("Remote target binary: %s\n", upgrade.TargetPath)
	if targetInput == "" && upgrade.RememberedTarget != "" {
		fmt.Printf("Used remembered upgrade target: %s\n", upgrade.RememberedTarget)
	}
	if upgrade.Service != nil {
		fmt.Printf("Started service %s (%s)\n", upgrade.Service.ID, displayOrDash(upgrade.Service.Name))
		fmt.Printf("Status: %s  PID: %s\n", displayOrDash(upgrade.Service.Status), formatOptionalInt(upgrade.Service.PID))
	}
	return nil
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
