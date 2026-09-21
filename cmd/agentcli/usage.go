package agentcli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/xhd2015/ai-critic/client"
	"github.com/xhd2015/less-gen/flags"
)

const usageHelpTemplate = `Usage: %[1]s usage <subcommand> [args...]

Manage the usage items shown in the macOS menu bar and Home page. Items live on
the server; the menu-bar app renders the server's title and dropdown text.

Subcommands:
  list                 List items with default marker, status, and summary
  add                  Add an item (--kind, --label, --home, --default)
  update               Change label, kind, home, api-url, or enabled state
  remove               Delete an item
  default              Pick the menu-bar item, or --rotate over enabled items
  show [<id>]          Print the full usage panel for one item or all items

Options:
  --json               Machine-readable output (list, show)
  --cached             Use cached values instead of fetching fresh (list, show)
  -h, --help           Show this help

Kinds: grok, codex, commandcode (commandcode requires --home).

Run '%[1]s usage <subcommand> -h' for subcommand options.
`

const usageListHelpTemplate = `Usage: %[1]s usage list [--json] [--cached]

List usage items in registry order with the menu-bar default marker, fetch
status, and the summary shown in the dropdown.

By default each enabled item fetches fresh usage from its provider before
printing; --cached prints the server's last cached values instead.

Options:
  --cached        Use cached values instead of fetching fresh.
  --json          Print the raw items JSON.
  -h, --help      Show this help.
`

const usageAddHelpTemplate = `Usage: %[1]s usage add --kind KIND [--label LABEL] [--id ID] [options]

Add a usage item. With --id omitted the id derives from --label, and with
--label omitted the label derives from the kind.

Options:
  --kind KIND       grok | codex | commandcode (required).
  --label LABEL     Menu text prefix, e.g. "CC v1".
  --id ID           Stable id; defaults to the slug of --label.
  --home DIR        Provider config dir; required for commandcode.
  --api-url URL     Override the provider API base URL.
  --default         Make this the menu-bar item (turns rotation off).
  --disabled        Register the item but keep it out of the menu bar.
  --no-validate     Skip the provider fetch that validates the item.
  --strict          Fail instead of warning when the provider fetch fails.
  -h, --help        Show this help.

Examples:
  %[1]s usage add --id cc-v1 --kind commandcode --label "CC v1" --home ~/.sandbox/commandcode-v1/.commandcode
`

const usageUpdateHelpTemplate = `Usage: %[1]s usage update <id> [options]

Change an existing usage item. At least one change flag is required.

Options:
  --label LABEL     New menu text prefix.
  --kind KIND       New provider kind.
  --home DIR        New provider config dir.
  --api-url URL     New provider API base URL.
  --enable          Include the item in the menu bar and rotation.
  --disable         Keep the registered item out of the menu bar.
  --no-validate     Skip the provider fetch when the binding changes.
  --strict          Fail instead of warning when the provider fetch fails.
  -h, --help        Show this help.
`

const usageRemoveHelpTemplate = `Usage: %[1]s usage remove <id>

Delete a usage item. When the removed item was the menu-bar default, the
server falls back to the next enabled item and the CLI prints a warning.
`

const usageDefaultHelpTemplate = `Usage: %[1]s usage default <id>
       %[1]s usage default --rotate [--start <id>]

Choose what the menu bar shows:

  <id>              Show that item, and turn rotation off.
  --rotate          Rotate the enabled items every 60 seconds.
  --start <id>      With --rotate, the item shown first and after each cycle.
`

const usageShowHelpTemplate = `Usage: %[1]s usage show [<id>] [--json] [--cached]

Print the usage text the menu bar renders: the full provider panel for one
item, or the dropdown line of every registered item when <id> is omitted.
Disabled items are printed too, so they can still be inspected.

By default each enabled item fetches fresh usage from its provider before
printing; --cached prints the server's last cached values instead.

Options:
  --cached        Use cached values instead of fetching fresh.
  --json          Print the raw items JSON.
  -h, --help      Show this help.
`

func usageCmdName() string {
	if active.Name != "" {
		return active.Name
	}
	return "local-agent"
}

func usageHelp() string {
	return fmt.Sprintf(usageHelpTemplate, usageCmdName())
}

func runUsage(resolve func() (*client.Client, error), args []string) error {
	if len(args) == 0 {
		fmt.Fprint(osStdout(), usageHelp())
		return nil
	}

	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list", "ls":
		return runUsageList(resolve, rest)
	case "add":
		return runUsageAdd(resolve, rest)
	case "update", "set":
		return runUsageUpdate(resolve, rest)
	case "remove", "rm", "delete", "del":
		return runUsageRemove(resolve, rest)
	case "default", "use":
		return runUsageDefault(resolve, rest)
	case "show":
		return runUsageShow(resolve, rest)
	case "-h", "--help":
		fmt.Fprint(osStdout(), usageHelp())
		return nil
	default:
		return fmt.Errorf("unknown usage subcommand: %s", sub)
	}
}

func runUsageList(resolve func() (*client.Client, error), args []string) error {
	var jsonOut, cached bool
	args, err := flags.New().
		Bool("--json", &jsonOut).
		Bool("--cached", &cached).
		Help("-h,--help", fmt.Sprintf(usageListHelpTemplate, usageCmdName())).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) > 0 {
		return fmt.Errorf("usage list takes no arguments, got %v", args)
	}

	cli, err := resolve()
	if err != nil {
		return err
	}
	resp, err := usageListItems(cli, cached)
	if err != nil {
		return cleanAPIError(err)
	}
	if jsonOut {
		return printUsageJSON(resp)
	}
	printUsageTable(resp)
	return nil
}

// usageListItems fetches the items view, refreshing from each provider unless
// the cached view was requested.
func usageListItems(cli *client.Client, cached bool) (*client.UsageItemsResponse, error) {
	if cached {
		return cli.ListUsageItems()
	}
	return cli.ListUsageItemsFresh()
}

func runUsageAdd(resolve func() (*client.Client, error), args []string) error {
	var id, label, kind, home, apiURL string
	var makeDefault, disabled, noValidate, strict bool

	args, err := flags.New().
		String("--id", &id).
		String("--label", &label).
		String("--kind", &kind).
		String("--home", &home).
		String("--api-url", &apiURL).
		Bool("--default", &makeDefault).
		Bool("--disabled", &disabled).
		Bool("--no-validate", &noValidate).
		Bool("--strict", &strict).
		Help("-h,--help", fmt.Sprintf(usageAddHelpTemplate, usageCmdName())).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) > 0 {
		return fmt.Errorf("usage add does not accept positional arguments: %v", args)
	}
	kind = strings.TrimSpace(kind)
	if kind == "" {
		return errors.New("usage add: --kind is required (grok, codex, commandcode)")
	}
	if kind == "commandcode" && strings.TrimSpace(home) == "" {
		return errors.New("usage add: --home is required for kind commandcode")
	}

	enabled := !disabled
	cli, err := resolve()
	if err != nil {
		return err
	}
	result, err := cli.AddUsageItem(client.UsageItemAddRequest{
		ID:           strings.TrimSpace(id),
		Label:        strings.TrimSpace(label),
		Kind:         kind,
		Home:         strings.TrimSpace(home),
		APIURL:       strings.TrimSpace(apiURL),
		Enabled:      &enabled,
		Default:      makeDefault,
		SkipValidate: noValidate,
		Strict:       strict,
	})
	if err != nil {
		return cleanAPIError(err)
	}
	printUsageWarnings(result.Warnings)
	if result.Item == nil {
		return nil
	}
	printUsageItemChange("Added", *result.Item)
	if !makeDefault {
		fmt.Fprintf(osStdout(), "Show it in the menu bar with: %s usage default %s\n", usageCmdName(), result.Item.ID)
	}
	return nil
}

func runUsageUpdate(resolve func() (*client.Client, error), args []string) error {
	var label, kind, home, apiURL string
	var labelSet, kindSet, homeSet, apiURLSet bool
	var enable, disable, noValidate, strict bool

	pre := flags.New().
		String("--label", &label).
		String("--kind", &kind).
		String("--home", &home).
		String("--api-url", &apiURL).
		Bool("--enable", &enable).
		Bool("--disable", &disable).
		Bool("--no-validate", &noValidate).
		Bool("--strict", &strict).
		Help("-h,--help", fmt.Sprintf(usageUpdateHelpTemplate, usageCmdName()))
	// Track which of the optional string flags were actually passed.
	rest, err := pre.Parse(markUsageFlagPresence(args, map[string]*bool{
		"--label":   &labelSet,
		"--kind":    &kindSet,
		"--home":    &homeSet,
		"--api-url": &apiURLSet,
	}))
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return fmt.Errorf("usage update requires exactly 1 argument <id>")
	}
	if !labelSet && !kindSet && !homeSet && !apiURLSet && !enable && !disable {
		return errors.New("usage update: no changes specified (use --label, --kind, --home, --api-url, --enable, --disable)")
	}
	if enable && disable {
		return errors.New("usage update: --enable and --disable are mutually exclusive")
	}

	req := client.UsageItemUpdateRequest{
		ID:           strings.TrimSpace(rest[0]),
		SkipValidate: noValidate,
		Strict:       strict,
	}
	if labelSet {
		req.Label = &label
	}
	if kindSet {
		req.Kind = &kind
	}
	if homeSet {
		req.Home = &home
	}
	if apiURLSet {
		req.APIURL = &apiURL
	}
	if enable {
		enabled := true
		req.Enabled = &enabled
	}
	if disable {
		enabled := false
		req.Enabled = &enabled
	}

	cli, err := resolve()
	if err != nil {
		return err
	}
	result, err := cli.UpdateUsageItem(req)
	if err != nil {
		return cleanAPIError(err)
	}
	printUsageWarnings(result.Warnings)
	if result.Item != nil {
		printUsageItemChange("Updated", *result.Item)
	}
	return nil
}

func runUsageRemove(resolve func() (*client.Client, error), args []string) error {
	args, err := flags.New().
		Help("-h,--help", fmt.Sprintf(usageRemoveHelpTemplate, usageCmdName())).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) != 1 {
		return fmt.Errorf("usage remove requires exactly 1 argument <id>")
	}

	cli, err := resolve()
	if err != nil {
		return err
	}
	result, err := cli.RemoveUsageItem(strings.TrimSpace(args[0]))
	if err != nil {
		return cleanAPIError(err)
	}
	printUsageWarnings(result.Warnings)
	fmt.Fprintf(osStdout(), "Removed usage item %s\n", result.Removed)
	return nil
}

func runUsageDefault(resolve func() (*client.Client, error), args []string) error {
	var rotate bool
	var start string

	args, err := flags.New().
		Bool("--rotate", &rotate).
		String("--start", &start).
		Help("-h,--help", fmt.Sprintf(usageDefaultHelpTemplate, usageCmdName())).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) > 1 {
		return fmt.Errorf("usage default takes at most 1 argument <id>, got %v", args)
	}
	id := strings.TrimSpace(start)
	if len(args) == 1 {
		if id != "" && id != strings.TrimSpace(args[0]) {
			return fmt.Errorf("usage default: <id> and --start disagree: %s vs %s", args[0], start)
		}
		id = strings.TrimSpace(args[0])
	}
	if !rotate && id == "" {
		return errors.New("usage default: pass <id>, or --rotate to rotate the enabled items")
	}

	cli, err := resolve()
	if err != nil {
		return err
	}
	before, err := cli.ListUsageItems()
	if err != nil {
		return cleanAPIError(err)
	}
	resp, err := cli.SetUsageDefault(id, rotate)
	if err != nil {
		return cleanAPIError(err)
	}

	if resp.Rotate {
		fmt.Fprintf(osStdout(), "Menu bar default: rotating over %s (60s), starting at %s\n",
			plural(len(enabledUsageIDs(resp)), "enabled item"), displayOrDefault(resp.Default))
		return nil
	}
	if before.Default != "" && before.Default != resp.Default {
		fmt.Fprintf(osStdout(), "Menu bar default: %s → %s (rotate off)\n", before.Default, resp.Default)
		return nil
	}
	fmt.Fprintf(osStdout(), "Menu bar default: %s (rotate off)\n", displayOrDefault(resp.Default))
	return nil
}

func runUsageShow(resolve func() (*client.Client, error), args []string) error {
	var jsonOut, cached bool
	args, err := flags.New().
		Bool("--json", &jsonOut).
		Bool("--cached", &cached).
		Help("-h,--help", fmt.Sprintf(usageShowHelpTemplate, usageCmdName())).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) > 1 {
		return fmt.Errorf("usage show takes at most 1 argument <id>, got %v", args)
	}

	cli, err := resolve()
	if err != nil {
		return err
	}
	resp, err := usageListItems(cli, cached)
	if err != nil {
		return cleanAPIError(err)
	}

	if len(args) == 0 {
		if jsonOut {
			return printUsageJSON(resp)
		}
		for i, item := range resp.Items {
			if i > 0 {
				fmt.Fprintln(osStdout())
			}
			fmt.Fprintf(osStdout(), "%s\n", item.Dropdown)
		}
		return nil
	}

	id := strings.TrimSpace(args[0])
	item := findUsageItem(resp, id)
	if item == nil {
		return fmt.Errorf("unknown usage item %q; known items: %s", id, usageIDs(resp))
	}
	if jsonOut {
		return printUsageJSON(item)
	}
	if strings.TrimSpace(item.Detail) != "" {
		fmt.Fprintf(osStdout(), "%s\n", item.Detail)
		return nil
	}
	fmt.Fprintf(osStdout(), "%s\n", item.Dropdown)
	return nil
}

func printUsageTable(resp *client.UsageItemsResponse) {
	out := osStdout()
	headers := []string{"ID", "DEF", "LABEL", "KIND", "ENABLED", "STATUS", "SUMMARY"}
	rows := make([][]string, 0, len(resp.Items))
	for _, item := range resp.Items {
		def := ""
		if !resp.Rotate && item.ID == resp.Default {
			def = "*"
		}
		rows = append(rows, []string{
			item.ID,
			def,
			item.Label,
			item.Kind,
			yesNo(item.Enabled),
			item.Status,
			usageSummary(item),
		})
	}

	widths := usageColumnWidths(headers, rows)
	fmt.Fprintf(out, "  %s\n", usageRow(headers, widths))
	for _, row := range rows {
		fmt.Fprintf(out, "  %s\n", usageRow(row, widths))
	}
	fmt.Fprintf(out, "\n%s · default %s · rotate %s\n",
		plural(len(rows), "item"), displayOrDefault(resp.Default), usageRotateLabel(resp.Rotate))
}

func usageColumnWidths(headers []string, rows [][]string) []int {
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
	return widths
}

func usageRow(cells []string, widths []int) string {
	parts := make([]string, len(cells))
	for i, cell := range cells {
		if i == len(cells)-1 {
			parts[i] = cell
			continue
		}
		parts[i] = cell + strings.Repeat(" ", widths[i]-len(cell)+2)
	}
	return strings.TrimRight(strings.Join(parts, ""), " ")
}

func usageSummary(item client.UsageItemView) string {
	prefix := item.Label + ": "
	if strings.HasPrefix(item.Dropdown, prefix) {
		return strings.TrimPrefix(item.Dropdown, prefix)
	}
	return strings.TrimSpace(item.Dropdown)
}

func usageRotateLabel(rotate bool) string {
	if rotate {
		return "on (60s)"
	}
	return "off"
}

func printUsageItemChange(verb string, item client.UsageItemView) {
	out := osStdout()
	fmt.Fprintf(out, "%s usage item %s (%s)\n", verb, item.ID, item.Kind)
	fmt.Fprintf(out, "  Label:   %s\n", item.Label)
	if item.Home != "" {
		fmt.Fprintf(out, "  Home:    %s\n", item.Home)
	}
	if item.APIURL != "" {
		fmt.Fprintf(out, "  API URL: %s\n", item.APIURL)
	}
	status := item.Status
	if summary := usageSummary(item); summary != "" {
		status += " · " + summary
	}
	fmt.Fprintf(out, "  Status:  %s\n", status)
}

func printUsageWarnings(warnings []string) {
	for _, warning := range warnings {
		fmt.Fprintf(os.Stderr, "warning: %s\n", warning)
	}
}

func printUsageJSON(value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("format usage json: %w", err)
	}
	fmt.Fprintln(osStdout(), string(data))
	return nil
}

func findUsageItem(resp *client.UsageItemsResponse, id string) *client.UsageItemView {
	for i := range resp.Items {
		if resp.Items[i].ID == id {
			return &resp.Items[i]
		}
	}
	return nil
}

func usageIDs(resp *client.UsageItemsResponse) string {
	if len(resp.Items) == 0 {
		return "(none)"
	}
	ids := make([]string, 0, len(resp.Items))
	for _, item := range resp.Items {
		ids = append(ids, item.ID)
	}
	return strings.Join(ids, ", ")
}

func enabledUsageIDs(resp *client.UsageItemsResponse) []string {
	var ids []string
	for _, item := range resp.Items {
		if item.Enabled {
			ids = append(ids, item.ID)
		}
	}
	return ids
}

func displayOrDefault(id string) string {
	if strings.TrimSpace(id) == "" {
		return "(none)"
	}
	return id
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func plural(count int, noun string) string {
	if count == 1 {
		return fmt.Sprintf("%d %s", count, noun)
	}
	return fmt.Sprintf("%d %ss", count, noun)
}

// Server errors arrive as "<status>: <message>" (e.g. "409 Conflict: usage item
// \"grok\" already exists"). This command prints the message alone so its own
// output stays readable; other CLI commands keep the raw error.
var apiStatusPrefix = regexp.MustCompile(`^[0-9]{3} [A-Za-z][A-Za-z ]*: `)

func cleanAPIError(err error) error {
	if err == nil {
		return nil
	}
	return errors.New(apiStatusPrefix.ReplaceAllString(err.Error(), ""))
}

// markUsageFlagPresence records which optional string flags appear in args, so
// update can tell "not passed" from "passed empty".
func markUsageFlagPresence(args []string, seen map[string]*bool) []string {
	for _, arg := range args {
		name := arg
		if idx := strings.Index(arg, "="); idx >= 0 {
			name = arg[:idx]
		}
		if target, ok := seen[name]; ok {
			*target = true
		}
	}
	return args
}
