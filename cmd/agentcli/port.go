package agentcli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/xhd2015/ai-critic/client"
	"github.com/xhd2015/less-gen/flags"
)

const portHelp = `Usage: remote-agent port <subcommand> [args...]

List remote listening ports and open short-lived public visits.

Subcommands:
  list [--json] [--forwards]
      List ports listening on the remote machine (PORT / PID / COMMAND).

  visit <port> [--provider auto|owned|quick] [--idle 10m] [--json] [--detach]
      Open an ad-hoc public URL for a local port (idle reverse-proxy hop).

  visit list
      List active ad-hoc visits.

  visit stop <id|port>
      Stop an active ad-hoc visit by session id or local port.
`

const portListHelp = `Usage: remote-agent port list [--json] [--forwards]

List TCP ports currently listening on the remote server.

Options:
  --json       Emit a JSON array (no ANSI).
  --forwards   Also include active persistent port forwards.
  -h, --help   Show this help message.
`

const portVisitHelp = `Usage: remote-agent port visit <port> [options...]
       remote-agent port visit list
       remote-agent port visit stop <id|port>

Open a short-lived public mapping for a remote local port via Cloudflare
(owned domain when available, else trycloudflare). Traffic is routed through
an idle reverse-proxy hop that shuts down after inactivity.

Options:
  --provider auto|owned|quick   Provider selection (default: auto).
  --idle DURATION               Idle timeout (default: 10m). Accepts 100ms, 30s, 10m.
  --json                        Machine-readable JSON on stdout (no ANSI).
  --detach                      Start and exit; leave the visit running on the server.
  -h, --help                    Show this help message.
`

func runPort(resolve func() (*client.Client, error), args []string) error {
	if len(args) == 0 {
		fmt.Print(portHelp)
		return nil
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list":
		return runPortList(resolve, rest)
	case "visit":
		return runPortVisit(resolve, rest)
	case "-h", "--help":
		fmt.Print(portHelp)
		return nil
	default:
		return fmt.Errorf("unknown port subcommand: %s", sub)
	}
}

func runPortList(resolve func() (*client.Client, error), args []string) error {
	var jsonOut bool
	var withForwards bool
	args, err := flags.
		Bool("--json", &jsonOut).
		Bool("--forwards", &withForwards).
		Help("-h,--help", portListHelp).
		HelpNoExit().
		Parse(args)
	if err != nil {
		if errors.Is(err, flags.ErrHelp) {
			return nil
		}
		return err
	}
	if len(args) > 0 {
		return fmt.Errorf("port list takes no arguments, got %v", args)
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	ports, err := cli.ListLocalPorts()
	if err != nil {
		return err
	}

	var forwards []client.PortForwardInfo
	if withForwards {
		forwards, err = cli.ListPortForwards()
		if err != nil {
			return err
		}
	}

	if jsonOut {
		type row map[string]interface{}
		out := make([]row, 0, len(ports)+len(forwards))
		for _, p := range ports {
			out = append(out, row{
				"port":    p.Port,
				"pid":     p.PID,
				"ppid":    p.PPID,
				"command": p.Command,
				"cmdline": p.Cmdline,
				"type":    "local",
			})
		}
		for _, f := range forwards {
			out = append(out, row{
				"port":      f.LocalPort,
				"localPort": f.LocalPort,
				"label":     f.Label,
				"publicUrl": f.PublicURL,
				"status":    f.Status,
				"provider":  f.Provider,
				"type":      f.Type,
			})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		return enc.Encode(out)
	}

	if len(ports) == 0 && len(forwards) == 0 {
		fmt.Println("No listening ports.")
		return nil
	}

	// Table: PORT PID COMMAND
	fmt.Printf("%-8s %-8s %s\n", "PORT", "PID", "COMMAND")
	for _, p := range ports {
		cmd := p.Command
		if cmd == "" {
			cmd = p.Cmdline
		}
		if cmd == "" {
			cmd = "-"
		}
		fmt.Printf("%-8d %-8d %s\n", p.Port, p.PID, cmd)
	}
	if withForwards && len(forwards) > 0 {
		if len(ports) > 0 {
			fmt.Println()
		}
		fmt.Printf("%-8s %-12s %s\n", "PORT", "STATUS", "PUBLIC URL")
		for _, f := range forwards {
			url := f.PublicURL
			if url == "" {
				url = "-"
			}
			fmt.Printf("%-8d %-12s %s\n", f.LocalPort, f.Status, url)
		}
	}
	return nil
}

func runPortVisit(resolve func() (*client.Client, error), args []string) error {
	if len(args) == 0 {
		fmt.Print(portVisitHelp)
		return nil
	}
	if args[0] == "-h" || args[0] == "--help" {
		fmt.Print(portVisitHelp)
		return nil
	}
	if args[0] == "list" {
		return runPortVisitList(resolve, args[1:])
	}
	if args[0] == "stop" {
		return runPortVisitStop(resolve, args[1:])
	}

	var provider string
	var idleStr string
	var jsonOut bool
	var detach bool
	args, err := flags.
		String("--provider", &provider).
		String("--idle", &idleStr).
		Bool("--json", &jsonOut).
		Bool("--detach", &detach).
		Help("-h,--help", portVisitHelp).
		HelpNoExit().
		Parse(args)
	if err != nil {
		if errors.Is(err, flags.ErrHelp) {
			return nil
		}
		return err
	}
	if len(args) == 0 {
		return fmt.Errorf("port visit requires a port number")
	}
	if len(args) > 1 {
		return fmt.Errorf("unexpected arguments: %v", args[1:])
	}

	port, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid port: %s", args[0])
	}
	if port <= 0 || port > 65535 {
		return fmt.Errorf("invalid port: %d", port)
	}

	var idle time.Duration
	if idleStr != "" {
		idle, err = time.ParseDuration(idleStr)
		if err != nil {
			return fmt.Errorf("invalid --idle duration %q: %w", idleStr, err)
		}
		if idle <= 0 {
			return fmt.Errorf("invalid --idle duration %q: must be positive", idleStr)
		}
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	visit, err := cli.StartPortVisit(port, provider, idle)
	if err != nil {
		return err
	}

	if !visit.Listening {
		fmt.Fprintf(os.Stderr, "warning: port %d is not listening\n", port)
	}

	if jsonOut {
		out := map[string]interface{}{
			"id":           visit.ID,
			"port":         visit.Port,
			"public_url":   visit.PublicURL,
			"provider":     visit.Provider,
			"idle_seconds": visit.IdleSeconds,
			"status":       visit.Status,
		}
		if visit.ProxyPort > 0 {
			out["proxy_port"] = visit.ProxyPort
		}
		if visit.Hostname != "" {
			out["hostname"] = visit.Hostname
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(out); err != nil {
			return err
		}
	} else {
		fmt.Printf("Visit started\n")
		fmt.Printf("  Provider: %s\n", visit.Provider)
		fmt.Printf("  URL:      %s\n", visit.PublicURL)
		fmt.Printf("  Port:     %d\n", visit.Port)
		fmt.Printf("  Idle:     %s\n", formatIdle(visit.IdleSeconds))
		fmt.Printf("  ID:       %s\n", visit.ID)
	}

	if detach {
		return nil
	}

	// Foreground: wait until the session is gone (idle expiry or remote stop).
	return waitVisitGone(cli, visit.ID, idle)
}

func runPortVisitList(resolve func() (*client.Client, error), args []string) error {
	var jsonOut bool
	args, err := flags.
		Bool("--json", &jsonOut).
		Help("-h,--help", portVisitHelp).
		HelpNoExit().
		Parse(args)
	if err != nil {
		if errors.Is(err, flags.ErrHelp) {
			return nil
		}
		return err
	}
	if len(args) > 0 {
		return fmt.Errorf("visit list takes no arguments, got %v", args)
	}

	cli, err := resolve()
	if err != nil {
		return err
	}
	list, err := cli.ListPortVisits()
	if err != nil {
		return err
	}

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		if list == nil {
			list = []client.PortVisit{}
		}
		return enc.Encode(list)
	}

	if len(list) == 0 {
		fmt.Println("No active visits.")
		return nil
	}
	fmt.Printf("%-12s %-8s %-18s %s\n", "ID", "PORT", "PROVIDER", "PUBLIC URL")
	for _, v := range list {
		fmt.Printf("%-12s %-8d %-18s %s\n", v.ID, v.Port, v.Provider, v.PublicURL)
	}
	return nil
}

func runPortVisitStop(resolve func() (*client.Client, error), args []string) error {
	args, err := flags.
		Help("-h,--help", portVisitHelp).
		HelpNoExit().
		Parse(args)
	if err != nil {
		if errors.Is(err, flags.ErrHelp) {
			return nil
		}
		return err
	}
	if len(args) != 1 {
		return fmt.Errorf("visit stop requires <id|port>")
	}
	cli, err := resolve()
	if err != nil {
		return err
	}
	if err := cli.StopPortVisit(args[0]); err != nil {
		return err
	}
	fmt.Printf("Stopped visit %s\n", args[0])
	return nil
}

func waitVisitGone(cli *client.Client, id string, idle time.Duration) error {
	// Poll until the session disappears. Cap wait by idle + margin.
	deadline := time.Now().Add(idle + 30*time.Second)
	if idle <= 0 {
		deadline = time.Now().Add(11 * time.Minute)
	}
	// For very short idle (e.g. 100ms), poll frequently.
	interval := 50 * time.Millisecond
	if idle > time.Second {
		interval = 200 * time.Millisecond
	}
	for time.Now().Before(deadline) {
		list, err := cli.ListPortVisits()
		if err != nil {
			return err
		}
		found := false
		for _, v := range list {
			if v.ID == id {
				found = true
				break
			}
		}
		if !found {
			return nil
		}
		time.Sleep(interval)
	}
	// Timed out waiting; still success if session may linger (detach semantics no).
	// Foreground treats remaining session as soft end — return nil after wait.
	return nil
}

func formatIdle(seconds float64) string {
	if seconds <= 0 {
		return "10m (default)"
	}
	d := time.Duration(seconds * float64(time.Second))
	return d.String()
}
