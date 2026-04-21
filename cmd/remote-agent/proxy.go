package main

import (
	"fmt"
	"strings"

	"github.com/xhd2015/lifelog-private/ai-critic/client"
)

const proxyHelp = `Usage: remote-agent proxy <subcommand> [args...]

Inspect proxy servers configured in the remote server's settings.

Subcommands:
  list
      List all configured proxy servers. Passwords are masked.
`

const proxyListHelp = `Usage: remote-agent proxy list

List all proxy servers configured in the remote server's settings. The
output includes each proxy's ID, name, URL, matching domains and
credentials (passwords are masked).
`

func runProxy(resolve func() (*client.Client, error), args []string) error {
	if len(args) == 0 {
		fmt.Print(proxyHelp)
		return nil
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list":
		return runProxyList(resolve, rest)
	case "-h", "--help":
		fmt.Print(proxyHelp)
		return nil
	default:
		return fmt.Errorf("unknown proxy subcommand: %s", sub)
	}
}

func runProxyList(resolve func() (*client.Client, error), args []string) error {
	if len(args) > 0 {
		if args[0] == "-h" || args[0] == "--help" {
			fmt.Print(proxyListHelp)
			return nil
		}
		return fmt.Errorf("proxy list takes no arguments, got %v", args)
	}

	cli, err := resolve()
	if err != nil {
		return err
	}
	servers, err := cli.ListProxies()
	if err != nil {
		return err
	}

	if len(servers) == 0 {
		fmt.Println("No proxy servers configured.")
		return nil
	}

	// Render as a simple aligned table. We pre-compute per-column widths
	// so the output stays readable even when names, hosts or domains
	// vary in length across rows.
	headers := []string{"ID", "NAME", "URL", "USER", "PASSWORD", "DOMAINS"}
	rows := make([][]string, 0, len(servers))
	for _, s := range servers {
		rows = append(rows, []string{
			s.ID,
			s.Name,
			proxyURL(s),
			s.Username,
			maskPassword(s.Password),
			strings.Join(s.Domains, ","),
		})
	}

	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	printRow(headers, widths)
	for _, row := range rows {
		printRow(row, widths)
	}
	return nil
}

// proxyURL formats a proxy's host/port/protocol as a single URL-ish
// string suitable for a single-row table cell.
func proxyURL(s client.ProxyServer) string {
	protocol := s.Protocol
	if protocol == "" {
		protocol = "http"
	}
	return fmt.Sprintf("%s://%s:%d", protocol, s.Host, s.Port)
}

// maskPassword returns a fixed-length mask so the table never reveals
// the real password length. Empty passwords render as a single dash so
// the column stays populated and distinguishable.
func maskPassword(pw string) string {
	if pw == "" {
		return "-"
	}
	return "********"
}

func printRow(cells []string, widths []int) {
	parts := make([]string, len(cells))
	for i, c := range cells {
		parts[i] = fmt.Sprintf("%-*s", widths[i], c)
	}
	fmt.Println(strings.Join(parts, "  "))
}
