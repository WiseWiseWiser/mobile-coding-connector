package agentcli

import (
	"fmt"
	"os"
	"strings"

	"github.com/xhd2015/ai-critic/client"
	"github.com/xhd2015/ai-critic/cmd/agentcli/streamcmd"
	"github.com/xhd2015/ai-critic/server/cloudflared"
)

const cloudflaredHelp = `Usage: %s cloudflared <command>

Origin cloudflared backend. Exactly one of native, qemu, proxy.

Commands:
  status                 Show configured backend and what is actually serving
  use native|qemu|proxy  Switch backend, republish, then stop the previous one

Run '%s cloudflared use -h' for backends.
`

const cloudflaredUseHelp = `Usage: %s cloudflared use native|qemu|proxy

  native   host cloudflared on this origin
  qemu     cloudflared inside the qemu guest
  proxy    edge cloudflare-proxy; this origin does not run cloudflared
           (proxy_url and token from ~/.ai-critic/cloudflare.json)
`

func runCloudflared(resolve func() (*client.Client, error), args []string) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		fmt.Fprintf(osStdout(), cloudflaredHelp, active.Name, active.Name)
		return nil
	}
	switch args[0] {
	case "status":
		return runCloudflaredStatus(resolve, args[1:])
	case "use":
		return runCloudflaredUse(resolve, args[1:])
	default:
		return fmt.Errorf("unknown cloudflared command %q", args[0])
	}
}

func runCloudflaredStatus(resolve func() (*client.Client, error), args []string) error {
	if len(args) > 0 {
		if isHelpToken(args[0]) {
			fmt.Fprintf(osStdout(), "Usage: %s cloudflared status\n\nPrint each fact as soon as it is known. backend is last.\n", active.Name)
			return nil
		}
		return fmt.Errorf("cloudflared status does not take arguments")
	}
	return streamcmd.Run(resolve, streamcmd.Spec{
		Method:  "GET",
		Path:    "/api/remote-agent/cloudflared/status/stream",
		Printer: streamcmd.Printer{Log: printStreamLine},
	})
}

func printStreamLine(ev client.StreamEvent) error {
	if strings.HasPrefix(ev.Message, "warning:") {
		fmt.Fprintln(os.Stderr, ev.Message)
	} else {
		fmt.Fprintln(osStdout(), ev.Message)
	}
	_ = osStdout().Sync()
	return nil
}

func runCloudflaredUse(resolve func() (*client.Client, error), args []string) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		if len(args) == 0 {
			return fmt.Errorf("backend required: native, qemu, or proxy")
		}
		fmt.Fprintf(osStdout(), cloudflaredUseHelp, active.Name)
		return nil
	}
	if len(args) != 1 {
		return fmt.Errorf("usage: %s cloudflared use native|qemu|proxy", active.Name)
	}
	if _, err := cloudflared.ParseBackend(args[0]); err != nil {
		return err
	}
	return streamcmd.Run(resolve, streamcmd.Spec{
		Method:  "POST",
		Path:    "/api/remote-agent/cloudflared/use/stream",
		Body:    map[string]string{"backend": args[0]},
		Printer: streamcmd.Printer{Log: printStreamLine},
	})
}
