package wsproxy_singbox

import (
	"fmt"
	"os"

	"github.com/xhd2015/ai-critic/client"
)

func RunClientConfig(getClient func() (*client.Client, error), opts ClientConfigOptions) error {
	c, err := getClient()
	if err != nil {
		return err
	}

	vmess, err := currentHooks.FetchVMess(c)
	if err != nil {
		return err
	}

	// client-config shows the sing-box layer only; run-tun adds an xray sidecar.
	data, err := BuildSingBoxTunConfig(vmess, buildTunConfigOptions(0))
	if err != nil {
		return err
	}

	if opts.OutputFile != "" {
		if err := os.WriteFile(opts.OutputFile, data, 0600); err != nil {
			return fmt.Errorf("write output file: %w", err)
		}
		return nil
	}

	fmt.Print(string(data))
	return nil
}
