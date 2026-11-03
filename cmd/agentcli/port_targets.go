package agentcli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/xhd2015/ai-critic/client"
	"github.com/xhd2015/ai-critic/server/config"
	"github.com/xhd2015/less-gen/flags"
)

func hasOriginFlags(args []string) bool {
	for _, a := range args {
		if strings.HasPrefix(a, "--target-host") || strings.HasPrefix(a, "--target-scheme") || strings.HasPrefix(a, "map_") {
			return true
		}
	}
	return false
}
func runTargetMappingAdd(resolve func() (*client.Client, error), args []string, update bool) error {
	var host, scheme, hostname, provider, user, token string
	var jsonOut, requireAuth, clearAuth, shared bool
	args, err := flags.String("--target-host", &host).String("--target-scheme", &scheme).
		String("--hostname", &hostname).String("--provider", &provider).
		Bool("--require-auth", &requireAuth).Bool("--clear-auth", &clearAuth).
		Bool("--shared-token", &shared).String("--auth-user", &user).String("--auth-token", &token).
		Bool("--json", &jsonOut).Help("-h,--help", portMappingHelpText()).HelpNoExit().Parse(args)
	if errors.Is(err, flags.ErrHelp) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(args) != 1 {
		return fmt.Errorf("mapping requires <port> (add) or <id|port> (update)")
	}
	cli, err := resolve()
	if err != nil {
		return err
	}
	target := config.OriginTarget{}
	if update {
		saved, e := cli.SavedPortMappings()
		if e != nil {
			return e
		}
		var matches []client.PortMappingRecord
		if rec, ok := saved.Mappings[args[0]]; ok {
			matches = append(matches, rec)
		} else {
			p, _ := strconv.Atoi(args[0])
			for _, rec := range saved.Mappings {
				if rec.Target.Port == p && rec.Desired {
					matches = append(matches, rec)
				}
			}
		}
		if len(matches) != 1 {
			return fmt.Errorf("selector %s matches %d mappings; specify a mapping ID", args[0], len(matches))
		}
		t := matches[0].Target
		target = config.OriginTarget{Host: t.Host, Scheme: t.Scheme, Port: t.Port}
	} else {
		p, e := strconv.Atoi(args[0])
		if e != nil {
			return fmt.Errorf("invalid port %q", args[0])
		}
		target.Port = p
	}
	if host != "" {
		target.Host = host
	}
	if scheme != "" {
		target.Scheme = scheme
	}
	target, err = target.Normalize(0)
	if err != nil {
		return err
	}
	if !update && hostname == "" {
		return fmt.Errorf("mapping add requires --hostname")
	}
	if strings.ContainsAny(hostname, " /\r\n\t") {
		return fmt.Errorf("--hostname must be a bare public hostname")
	}
	body := map[string]any{"target": client.OriginTarget{Host: target.Host, Scheme: target.Scheme, Port: target.Port}, "port": target.Port}
	if hostname != "" {
		body["label"] = hostname
	}
	if !update || provider != "" {
		p, e := normalizePortProvider(provider)
		if e != nil {
			return e
		}
		body["provider"] = p
	}
	if requireAuth || clearAuth || shared || user != "" || token != "" {
		auth, e := resolvePortMappingAuth(false, nil, requireAuth, clearAuth, shared, user, token)
		if e != nil {
			return e
		}
		body["auth"] = auth
	}
	out, err := cli.WritePortMapping(update, args[0], body)
	if err != nil {
		return err
	}
	if out.Auth != nil {
		out.Auth.Token = ""
	}
	if out.Origin != nil && !out.Origin.TCPReachable {
		fmt.Fprintf(os.Stderr, "warning: target %s is unreachable from the server; mapping saved, requests may return 502 (%s)\n", target.URL(), out.Origin.Error)
	}
	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(out)
	}
	verb := "added"
	if update {
		verb = "updated"
	}
	fmt.Printf("mapping %s\n  ID:        %s\n  Target:    %s\n  URL:       %s\n  Tunnel:    %s\n", verb, out.ID, target.URL(), out.PublicURL, out.Status)
	if out.Auth != nil && out.Auth.Required {
		fmt.Println("  Auth:      required (token hidden; use mapping list --show-token)")
	}
	return nil
}
func runTargetMappingDelete(resolve func() (*client.Client, error), args []string) error {
	var forget, jsonOut bool
	args, err := flags.Bool("--forget", &forget).Bool("--json", &jsonOut).Help("-h,--help", portMappingHelpText()).HelpNoExit().Parse(args)
	if errors.Is(err, flags.ErrHelp) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(args) != 1 {
		return fmt.Errorf("mapping delete requires <id|port>")
	}
	cli, err := resolve()
	if err != nil {
		return err
	}
	if err = cli.RemovePortMapping(args[0], forget); err != nil {
		return err
	}
	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(map[string]any{"id": args[0], "deleted": true})
	}
	if forget {
		fmt.Printf("mapping deleted: %s (saved name and auth cleared)\n", args[0])
	} else {
		fmt.Printf("mapping deleted: %s (saved settings kept)\n", args[0])
	}
	return nil
}
func listTargetMappings(cli *client.Client, forwards []client.PortForwardInfo, auth map[string]client.PortAuth, jsonOut, showToken bool) error {
	saved, err := cli.SavedPortMappings()
	if err != nil {
		return err
	}
	type row struct {
		client.PortForwardInfo
		Desired bool   `json:"desired"`
		Source  string `json:"source"`
	}
	rows := []row{}
	seen := map[string]bool{}
	for _, f := range forwards {
		if f.ID != "" {
			seen[f.ID] = true
			f.Auth = mergeMappingAuth(f.Auth, auth[f.ID], showToken)
		}
		rows = append(rows, row{f, true, portForwardSource(f)})
	}
	for id, r := range saved.Mappings {
		if seen[id] {
			continue
		}
		status := "inactive"
		if r.Desired {
			status = "error"
		}
		target := r.Target
		rows = append(rows, row{client.PortForwardInfo{ID: id, Target: &target, LocalPort: target.Port, Label: r.Hostname, Provider: r.Provider, Status: status, Auth: mergeMappingAuth(&r.Auth, auth[id], showToken)}, r.Desired, "-"})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(rows)
	}
	fmt.Printf("%-30s %-42s %-12s %-9s %-13s %s\n", "ID", "TARGET", "STATUS", "AUTH", "SOURCE", "PUBLIC URL / SAVED HOSTNAME")
	for _, r := range rows {
		target := "http://127.0.0.1:" + strconv.Itoa(r.LocalPort)
		if r.Target != nil {
			target = r.Target.URL()
		}
		public := r.PublicURL
		if public == "" {
			public = r.Label
		}
		authLabel := "-"
		if r.Auth != nil && r.Auth.Required {
			authLabel = "required"
		}
		fmt.Printf("%-30s %-42s %-12s %-9s %-13s %s\n", r.ID, target, r.Status, authLabel, r.Source, public)
		if showToken && r.Auth != nil && r.Auth.Token != "" {
			fmt.Printf("  user: %s  token: %s\n", portMappingAuthUser(r.Auth.User), r.Auth.Token)
		}
	}
	return nil
}

func hasForgetFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--forget" {
			return true
		}
	}
	return false
}
