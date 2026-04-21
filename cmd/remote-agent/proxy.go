package main

import (
	"fmt"

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

	// Horizontal tables wrap poorly on narrow terminals because long
	// IDs, URLs and domain lists blow past 80 columns. Render each
	// proxy as a short vertical record instead, separated by a blank
	// line. This reads well at any terminal width.
	for i, s := range servers {
		if i > 0 {
			fmt.Println()
		}
		printProxy(s)
	}
	return nil
}

// printProxy writes one proxy as an aligned "Field: value" block. The
// field column is padded so colons line up; long lists (e.g. domains)
// are printed one-per-line under a "Domains:" header so they never
// cause horizontal wrapping.
func printProxy(s client.ProxyServer) {
	const labelWidth = 10
	label := func(name string) string {
		return fmt.Sprintf("  %-*s", labelWidth, name+":")
	}

	name := s.Name
	if name == "" {
		name = "(unnamed)"
	}
	fmt.Printf("%s %s\n", label("Name"), name)
	fmt.Printf("%s %s\n", label("ID"), s.ID)
	fmt.Printf("%s %s\n", label("URL"), proxyURL(s))
	if s.Username != "" || s.Password != "" {
		fmt.Printf("%s %s\n", label("User"), displayOrDash(s.Username))
		fmt.Printf("%s %s\n", label("Password"), maskPassword(s.Password))
	}
	if len(s.Domains) == 0 {
		fmt.Printf("%s (none)\n", label("Domains"))
		return
	}
	if len(s.Domains) == 1 {
		fmt.Printf("%s %s\n", label("Domains"), s.Domains[0])
		return
	}
	fmt.Printf("%s\n", label("Domains"))
	for _, d := range s.Domains {
		fmt.Printf("    - %s\n", d)
	}
}

// proxyURL formats a proxy's host/port/protocol as a single URL-ish
// string. Credentials are never shown — the Username/Password fields
// carry that information.
func proxyURL(s client.ProxyServer) string {
	protocol := s.Protocol
	if protocol == "" {
		protocol = "http"
	}
	return fmt.Sprintf("%s://%s:%d", protocol, s.Host, s.Port)
}

// maskPassword returns a fixed-length mask for a non-empty password so
// the output never reveals the real length, or "(none)" when unset.
func maskPassword(pw string) string {
	if pw == "" {
		return "(none)"
	}
	return "********"
}

func displayOrDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
