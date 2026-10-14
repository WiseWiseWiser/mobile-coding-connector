package agentcli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xhd2015/ai-critic/client"
	"github.com/xhd2015/less-gen/flags"
)

func portHelpText() string {
	return fmt.Sprintf(`Usage: %[1]s port <subcommand> [args...]

List listening ports, publish them, and open short-lived public visits.

Subcommands:
  list [--json] [--forwards]
      List ports listening on the server (PORT / PID / COMMAND).

  mapping add <port> --hostname H [--provider owned|quick] [--json]
  mapping list [--json]
  mapping update <port> --hostname H [--provider owned|quick] [--json]
  mapping delete <port> [--forget] [--json]
      Persistent public mapping for a port (survives restarts).

  visit <port> [--provider auto|owned|quick] [--idle 10m] [--json] [--detach]
      Open an ad-hoc public URL for a local port (idle reverse-proxy hop).

  visit list
      List active ad-hoc visits.

  visit stop <id|port>
      Stop an active ad-hoc visit by session id or local port.
`, active.Name)
}

func portListHelpText() string {
	return fmt.Sprintf(`Usage: %s port list [--json] [--forwards]

List TCP ports currently listening on the server.

Options:
  --json       Emit a JSON array (no ANSI).
  --forwards   Also include active persistent port forwards.
  -h, --help   Show this help message.
`, active.Name)
}

func portMappingHelpText() string {
	return fmt.Sprintf(`Usage: %[1]s port mapping <add|list|update|delete> [args...]

Publish a port at a hostname that survives restarts, unlike 'port visit'.

  %[1]s port mapping add <port> --hostname H [--provider owned|quick] [--json]
  %[1]s port mapping list [--json]
  %[1]s port mapping update <port> --hostname H [--provider owned|quick] [--json]
  %[1]s port mapping delete <port> [--forget] [--json]

Options:
  --hostname H   Public hostname, e.g. house-design-aes821.xhd2015.xyz.
  --provider P   owned (default) | quick | localtunnel, or a canonical server
                 name (cloudflare_owned, cloudflare_quick).
  --forget       delete: also clear the name saved for this port.
  --json         Machine-readable JSON on stdout (no ANSI).

Notes:
  A hostname already mapped to another port is taken over (the stale mapping is
  dropped), so 'delete' the other port first if that was not intended.
  'delete' keeps the saved name unless --forget is passed, so re-adding the same
  port at the same hostname needs no retyping.

  'list' shows a SOURCE column so two mappings are not mistaken for one another:
    forward        a plain port forward — added by hand, or by a service definition
                   (the two are indistinguishable, so neither is claimed)
    domain(boot)   a domain tunnel the server started for itself at boot
    domain         a domain tunnel requested explicitly
    exposed        an exposed URL
  A mapping whose name is saved but which is not serving shows '-' instead.
`, active.Name)
}

func portVisitHelpText() string {
	return fmt.Sprintf(`Usage: %[1]s port visit <port> [options...]
       %[1]s port visit list
       %[1]s port visit stop <id|port>

Open a short-lived public mapping for a port via Cloudflare
(owned domain when available, else trycloudflare). Traffic is routed through
an idle reverse-proxy hop that shuts down after inactivity.

Options:
  --provider auto|owned|quick   Provider selection (default: auto).
  --idle DURATION               Idle timeout (default: 10m). Accepts 100ms, 30s, 10m.
  --json                        Machine-readable JSON on stdout (no ANSI).
  --detach                      Start and exit; leave the visit running on the server.
  -h, --help                    Show this help message.
`, active.Name)
}

func runPort(resolve func() (*client.Client, error), args []string) error {
	if len(args) == 0 {
		fmt.Print(portHelpText())
		return nil
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list":
		return runPortList(resolve, rest)
	case "mapping":
		return runPortMapping(resolve, rest)
	case "visit":
		return runPortVisit(resolve, rest)
	case "-h", "--help":
		fmt.Print(portHelpText())
		return nil
	default:
		return fmt.Errorf("unknown port subcommand: %s", sub)
	}
}

// portMappingProviders maps the short names users type onto the names the
// server expects. A canonical name is passed through untouched so scripts that
// already speak the API keep working.
var portMappingProviders = map[string]string{
	"owned":       "cloudflare_owned",
	"quick":       "cloudflare_quick",
	"localtunnel": "localtunnel",
}

func normalizePortProvider(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "cloudflare_owned", nil
	}
	if canonical, ok := portMappingProviders[name]; ok {
		return canonical, nil
	}
	switch name {
	case "cloudflare_owned", "cloudflare_quick", "cloudflare_tunnel", "localtunnel":
		return name, nil
	}
	return "", fmt.Errorf("unknown provider: %s (want owned|quick|localtunnel or cloudflare_owned|cloudflare_quick)", name)
}

func runPortMapping(resolve func() (*client.Client, error), args []string) error {
	if len(args) == 0 {
		fmt.Print(portMappingHelpText())
		return nil
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "add":
		return runPortMappingAdd(resolve, rest, false)
	case "update":
		return runPortMappingAdd(resolve, rest, true)
	case "list":
		return runPortMappingList(resolve, rest)
	case "delete", "remove", "rm":
		return runPortMappingDelete(resolve, rest)
	case "-h", "--help":
		fmt.Print(portMappingHelpText())
		return nil
	default:
		return fmt.Errorf("unknown port mapping subcommand: %s", sub)
	}
}

// runPortMappingAdd publishes a port. update re-publishes one that is already
// mapped: the server refuses a second mapping on the same port, so the old one
// is removed first and restored if the new one fails.
func runPortMappingAdd(resolve func() (*client.Client, error), args []string, update bool) error {
	var hostname, provider string
	var jsonOut bool
	args, err := flags.
		String("--hostname", &hostname).
		String("--provider", &provider).
		Bool("--json", &jsonOut).
		Help("-h,--help", portMappingHelpText()).
		HelpNoExit().
		Parse(args)
	if err != nil {
		if errors.Is(err, flags.ErrHelp) {
			return nil
		}
		return err
	}
	verb := "add"
	if update {
		verb = "update"
	}
	if len(args) == 0 {
		return fmt.Errorf("port mapping %s requires a port number; see '%s port mapping --help'", verb, active.Name)
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
	hostname = strings.TrimSpace(hostname)
	if hostname == "" {
		return fmt.Errorf("port mapping %s requires --hostname", verb)
	}
	if strings.ContainsAny(hostname, " \t/") {
		return fmt.Errorf("invalid --hostname %q: want a bare hostname", hostname)
	}
	canonical, err := normalizePortProvider(provider)
	if err != nil {
		return err
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	var previous *client.PortForwardInfo
	if update {
		forwards, err := cli.ListPortForwards()
		if err != nil {
			return err
		}
		for i := range forwards {
			if forwards[i].LocalPort == port {
				previous = &forwards[i]
				break
			}
		}
		if previous == nil {
			return fmt.Errorf("port %d is not mapped; use 'port mapping add %d --hostname %s'", port, port, hostname)
		}
		if previous.Label == hostname {
			fmt.Printf("mapping unchanged\n")
			fmt.Printf("  Port:      %d\n", port)
			fmt.Printf("  Hostname:  %s\n", hostname)
			return nil
		}
		if err := cli.RemovePortForward(port); err != nil {
			return fmt.Errorf("remove old mapping: %w", err)
		}
	}

	// Remember the name first, mirroring the web UI, so the hostname survives
	// a later delete and can be reused without retyping.
	if err := cli.SetPortMappingName(port, hostname); err != nil {
		if update {
			restorePortMapping(cli, previous)
		}
		return fmt.Errorf("save mapping name: %w", err)
	}
	if err := cli.AddPortForward(port, hostname, canonical, "", ""); err != nil {
		if update {
			restorePortMapping(cli, previous)
		}
		if strings.Contains(err.Error(), "already being forwarded") {
			return fmt.Errorf("%w; use 'port mapping update %d --hostname %s'", err, port, hostname)
		}
		if strings.Contains(err.Error(), "409") {
			return fmt.Errorf("%w; run 'port mapping list' to find the port holding it", err)
		}
		return err
	}

	forward, err := findPortForward(cli, port)
	if err != nil {
		return err
	}
	if !portIsListening(cli, port) {
		fmt.Fprintf(os.Stderr, "warning: port %d is not listening; the mapping is published but will not serve until something binds it\n", port)
	}

	if jsonOut {
		out := map[string]interface{}{
			"port":       port,
			"hostname":   hostname,
			"provider":   canonical,
			"status":     forward.Status,
			"public_url": forward.PublicURL,
		}
		if previous != nil {
			out["previous_hostname"] = previous.Label
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		return enc.Encode(out)
	}

	fmt.Printf("mapping %s\n", map[bool]string{false: "added", true: "updated"}[update])
	fmt.Printf("  Port:      %d\n", port)
	if previous != nil {
		fmt.Printf("  Hostname:  %s   (was %s)\n", hostname, previous.Label)
	} else {
		fmt.Printf("  Hostname:  %s\n", hostname)
	}
	fmt.Printf("  Provider:  %s\n", canonical)
	if forward.PublicURL != "" {
		fmt.Printf("  URL:       %s\n", forward.PublicURL)
	}
	fmt.Printf("  Name:      saved (--forget on delete to clear)\n")
	return nil
}

// restorePortMapping best-effort puts the old mapping back after a failed
// update, so a typo cannot leave the port unpublished.
func restorePortMapping(cli *client.Client, previous *client.PortForwardInfo) {
	if previous == nil {
		return
	}
	if err := cli.AddPortForward(previous.LocalPort, previous.Label, previous.Provider, "", ""); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not restore %s on port %d: %v\n", previous.Label, previous.LocalPort, err)
	}
}

func findPortForward(cli *client.Client, port int) (*client.PortForwardInfo, error) {
	forwards, err := cli.ListPortForwards()
	if err != nil {
		return nil, err
	}
	for i := range forwards {
		if forwards[i].LocalPort == port {
			return &forwards[i], nil
		}
	}
	return &client.PortForwardInfo{LocalPort: port}, nil
}

func portIsListening(cli *client.Client, port int) bool {
	ports, err := cli.ListLocalPorts()
	if err != nil {
		// A failed probe must not fail a mapping that already succeeded.
		return true
	}
	for _, p := range ports {
		if p.Port == port {
			return true
		}
	}
	return false
}

// portForwardSource names where a forward came from, so two rows with different origins do not
// render as the same URL.
//
// It reports what the server already knows rather than inventing a field: `Type` distinguishes a
// plain forward from a domain/exposed/opencode tunnel, and `Bootstrap` separates a forward the
// server started for itself at boot from one somebody asked for. What it deliberately does NOT
// claim is "service" vs "manual" — the two are indistinguishable today, and naming one of them
// would be a guess printed as fact.
func portForwardSource(f client.PortForwardInfo) string {
	switch f.Type {
	case "", "port_forward":
		// A plain forward: added by hand, or by a service definition. Both are this type, so the
		// distinction is not knowable here and the column says only what is true.
		return "forward"
	case "domain":
		if f.Bootstrap {
			return "domain(boot)"
		}
		return "domain"
	case "exposed_url":
		return "exposed"
	default:
		return f.Type
	}
}

func runPortMappingList(resolve func() (*client.Client, error), args []string) error {
	var jsonOut bool
	args, err := flags.
		Bool("--json", &jsonOut).
		Help("-h,--help", portMappingHelpText()).
		HelpNoExit().
		Parse(args)
	if err != nil {
		if errors.Is(err, flags.ErrHelp) {
			return nil
		}
		return err
	}
	if len(args) > 0 {
		return fmt.Errorf("port mapping list takes no arguments, got %v", args)
	}

	cli, err := resolve()
	if err != nil {
		return err
	}
	forwards, err := cli.ListPortForwards()
	if err != nil {
		return err
	}
	names, err := cli.ListPortMappingNames()
	if err != nil {
		return err
	}

	type row struct {
		Port      int    `json:"port"`
		Source    string `json:"source,omitempty"`
		Status    string `json:"status"`
		Provider  string `json:"provider"`
		PublicURL string `json:"publicUrl"`
		SavedName string `json:"savedName,omitempty"`
	}
	rows := make([]row, 0, len(forwards))
	seen := map[int]bool{}
	for _, f := range forwards {
		seen[f.LocalPort] = true
		rows = append(rows, row{
			Port: f.LocalPort, Source: portForwardSource(f), Status: f.Status,
			Provider: f.Provider, PublicURL: f.PublicURL,
			SavedName: names[strconv.Itoa(f.LocalPort)],
		})
	}
	// A saved name with no live mapping is worth showing: that is exactly the
	// state after 'delete', and it is what re-add would reuse.
	for portStr, domain := range names {
		port, err := strconv.Atoi(portStr)
		if err != nil || seen[port] {
			continue
		}
		rows = append(rows, row{Port: port, Status: "inactive", SavedName: domain})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Port < rows[j].Port })

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		return enc.Encode(rows)
	}
	if len(rows) == 0 {
		fmt.Println("No port mappings.")
		return nil
	}
	fmt.Printf("%-7s %-13s %-9s %-18s %s\n", "PORT", "SOURCE", "STATUS", "PROVIDER", "PUBLIC URL")
	for _, r := range rows {
		url := r.PublicURL
		if url == "" {
			if r.SavedName != "" {
				url = "(saved name: " + r.SavedName + ")"
			} else {
				url = "-"
			}
		}
		provider := r.Provider
		if provider == "" {
			provider = "-"
		}
		source := r.Source
		if source == "" {
			// A saved name with no live mapping: nothing is serving it, so there is no origin.
			source = "-"
		}
		fmt.Printf("%-7d %-13s %-9s %-18s %s\n", r.Port, source, r.Status, provider, url)
	}
	fmt.Printf("\n%d mapping(s)\n", len(rows))
	return nil
}

func runPortMappingDelete(resolve func() (*client.Client, error), args []string) error {
	var forget, jsonOut bool
	args, err := flags.
		Bool("--forget", &forget).
		Bool("--json", &jsonOut).
		Help("-h,--help", portMappingHelpText()).
		HelpNoExit().
		Parse(args)
	if err != nil {
		if errors.Is(err, flags.ErrHelp) {
			return nil
		}
		return err
	}
	if len(args) != 1 {
		return fmt.Errorf("port mapping delete requires a port number; see '%s port mapping --help'", active.Name)
	}
	port, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid port: %s", args[0])
	}

	cli, err := resolve()
	if err != nil {
		return err
	}
	names, err := cli.ListPortMappingNames()
	if err != nil {
		return err
	}
	savedName := names[strconv.Itoa(port)]

	if err := cli.RemovePortForward(port); err != nil {
		return err
	}
	if forget {
		if err := cli.DeletePortMappingName(port); err != nil {
			return err
		}
		savedName = ""
	}

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		return enc.Encode(map[string]interface{}{
			"port": port, "deleted": true, "saved_name": savedName, "forgot": forget,
		})
	}
	fmt.Printf("mapping deleted\n")
	fmt.Printf("  Port:  %d\n", port)
	if forget {
		fmt.Printf("  Name:  cleared\n")
	} else if savedName != "" {
		fmt.Printf("  Name:  %s kept (--forget to clear)\n", savedName)
	}
	return nil
}

func runPortList(resolve func() (*client.Client, error), args []string) error {
	var jsonOut bool
	var withForwards bool
	args, err := flags.
		Bool("--json", &jsonOut).
		Bool("--forwards", &withForwards).
		Help("-h,--help", portListHelpText()).
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
		fmt.Print(portVisitHelpText())
		return nil
	}
	if args[0] == "-h" || args[0] == "--help" {
		fmt.Print(portVisitHelpText())
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
		Help("-h,--help", portVisitHelpText()).
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
		Help("-h,--help", portVisitHelpText()).
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
		Help("-h,--help", portVisitHelpText()).
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
