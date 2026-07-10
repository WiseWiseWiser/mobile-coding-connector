package agentcli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xhd2015/ai-critic/client"
	"github.com/xhd2015/less-gen/flags"
)

const cronHelp = `Usage: remote-agent cron <subcommand> [args...]

Manage global scheduled cron tasks on the server.

Schedule modes (mutually exclusive on add/update):
  --every DURATION     Interval (next fire = last finish + duration)
  --cron EXPR          5-field cron in LOCAL wall time → converted to UTC when safe
  --cron-utc EXPR      5-field cron already in UTC (pass-through)

Cron expressions on the server are always 5-field UTC.
If --cron cannot be converted safely (ranges, steps, lists, DST ambiguity),
the command errors and you must use --cron-utc.

Subcommands:
  list
  add    --name --command (--every | --cron | --cron-utc) [--working-dir] [--timeout 1h] [--disabled]
  update <name-or-id> …
  remove <name-or-id>
  enable|disable <name-or-id>
  run <name-or-id>
  logs [--lines N] <name-or-id>
  history <name-or-id>
`

const cronAddHelp = `Usage: remote-agent cron add --name NAME --command CMD (--every DUR | --cron EXPR | --cron-utc EXPR) [options]

Options:
  --name NAME            Task name (required)
  --command CMD          Shell command via bash -lc (required)
  --every DURATION       Interval schedule, e.g. 5m, 30s, 1h
  --cron EXPR            Local wall-time cron (safe convert to UTC)
  --cron-utc EXPR        UTC cron expression (pass-through)
  --working-dir DIR      Working directory
  --timeout DURATION     Kill after this long (default 1h; must be > 0)
  --disabled             Create disabled
  -h, --help             Show help
`

const cronLogsHelp = `Usage: remote-agent cron logs [--lines N] <name-or-id>

Stream one cron task's log file.
`

func runCron(resolve func() (*client.Client, error), args []string) error {
	if len(args) == 0 {
		fmt.Print(cronHelp)
		return nil
	}
	switch args[0] {
	case "list":
		return runCronList(resolve, args[1:])
	case "add":
		return runCronAdd(resolve, args[1:])
	case "update":
		return runCronUpdate(resolve, args[1:])
	case "remove", "delete", "rm":
		return runCronRemove(resolve, args[1:])
	case "enable":
		return runCronEnableDisable(resolve, true, args[1:])
	case "disable":
		return runCronEnableDisable(resolve, false, args[1:])
	case "run":
		return runCronRun(resolve, args[1:])
	case "logs":
		return runCronLogs(resolve, args[1:])
	case "history":
		return runCronHistory(resolve, args[1:])
	case "-h", "--help":
		fmt.Print(cronHelp)
		return nil
	default:
		return fmt.Errorf("unknown cron subcommand: %s", args[0])
	}
}

func runCronList(resolve func() (*client.Client, error), args []string) error {
	args, err := flags.Help("-h,--help", "Usage: remote-agent cron list\n").Parse(args)
	if err != nil {
		return err
	}
	if len(args) > 0 {
		return fmt.Errorf("cron list takes no positional arguments")
	}
	cli, err := resolve()
	if err != nil {
		return err
	}
	tasks, err := cli.ListCronTasks()
	if err != nil {
		return err
	}
	if len(tasks) == 0 {
		fmt.Println("No cron tasks found.")
		return nil
	}
	for i, task := range tasks {
		if i > 0 {
			fmt.Println()
		}
		printCronTask(task)
	}
	return nil
}

func runCronAdd(resolve func() (*client.Client, error), args []string) error {
	var (
		name, command, workingDir string
		every, cronLocal, cronUTC string
		timeout                   string
		disabled                  bool
	)
	args, err := flags.
		String("--name", &name).
		String("--command", &command).
		String("--every", &every).
		String("--cron", &cronLocal).
		String("--cron-utc", &cronUTC).
		String("--working-dir", &workingDir).
		String("--timeout", &timeout).
		Bool("--disabled", &disabled).
		Help("-h,--help", cronAddHelp).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) > 0 {
		return fmt.Errorf("unexpected arguments: %v", args)
	}
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("--name is required")
	}
	if strings.TrimSpace(command) == "" {
		return fmt.Errorf("--command is required")
	}

	mode, interval, cronExpr, err := resolveCronScheduleFlags(every, cronLocal, cronUTC)
	if err != nil {
		return err
	}

	def := client.CronTaskDefinition{
		Name:         name,
		Command:      command,
		WorkingDir:   workingDir,
		ScheduleMode: mode,
		Interval:     interval,
		CronExpr:     cronExpr,
		Timeout:      timeout,
	}
	if disabled {
		f := false
		def.Enabled = &f
	}

	cli, err := resolve()
	if err != nil {
		return err
	}
	st, err := cli.CreateCronTask(def)
	if err != nil {
		return err
	}

	// On successful --cron convert, print both local and stored UTC.
	if strings.TrimSpace(cronLocal) != "" {
		fmt.Printf("local cron: %s\n", strings.TrimSpace(cronLocal))
		fmt.Printf("stored UTC: %s\n", st.CronExpr)
	}
	fmt.Printf("Created cron task %s (%s)\n", st.ID, displayOrDash(st.Name))
	printCronTask(*st)
	return nil
}

func runCronUpdate(resolve func() (*client.Client, error), args []string) error {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		fmt.Println("Usage: remote-agent cron update <name-or-id> [--name] [--command] [--every|--cron|--cron-utc] [--timeout] [--working-dir]")
		return nil
	}
	target := args[0]
	rest := args[1:]
	var (
		name, command, workingDir string
		every, cronLocal, cronUTC string
		timeout                   string
		setDisabled, setEnabled   bool
	)
	rest, err := flags.
		String("--name", &name).
		String("--command", &command).
		String("--every", &every).
		String("--cron", &cronLocal).
		String("--cron-utc", &cronUTC).
		String("--working-dir", &workingDir).
		String("--timeout", &timeout).
		Bool("--disabled", &setDisabled).
		Bool("--enabled", &setEnabled).
		Help("-h,--help", "Usage: remote-agent cron update <name-or-id> [options]\n").
		Parse(rest)
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return fmt.Errorf("unexpected arguments: %v", rest)
	}

	cli, err := resolve()
	if err != nil {
		return err
	}
	task, err := resolveCronTarget(cli, target)
	if err != nil {
		return err
	}

	def := client.CronTaskDefinition{
		ID:           task.ID,
		Name:         task.Name,
		Command:      task.Command,
		WorkingDir:   task.WorkingDir,
		ScheduleMode: task.ScheduleMode,
		Interval:     task.Interval,
		CronExpr:     task.CronExpr,
		Timeout:      task.Timeout,
	}
	if name != "" {
		def.Name = name
	}
	if command != "" {
		def.Command = command
	}
	if workingDir != "" {
		def.WorkingDir = workingDir
	}
	if timeout != "" {
		def.Timeout = timeout
	}
	if every != "" || cronLocal != "" || cronUTC != "" {
		mode, interval, cronExpr, err := resolveCronScheduleFlags(every, cronLocal, cronUTC)
		if err != nil {
			return err
		}
		def.ScheduleMode = mode
		def.Interval = interval
		def.CronExpr = cronExpr
	}
	if setDisabled {
		f := false
		def.Enabled = &f
	} else if setEnabled {
		t := true
		def.Enabled = &t
	}

	st, err := cli.UpdateCronTask(def)
	if err != nil {
		return err
	}
	if strings.TrimSpace(cronLocal) != "" {
		fmt.Printf("local cron: %s\n", strings.TrimSpace(cronLocal))
		fmt.Printf("stored UTC: %s\n", st.CronExpr)
	}
	fmt.Printf("Updated cron task %s (%s)\n", st.ID, displayOrDash(st.Name))
	printCronTask(*st)
	return nil
}

func runCronRemove(resolve func() (*client.Client, error), args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("cron remove requires exactly 1 argument <name-or-id>")
	}
	cli, err := resolve()
	if err != nil {
		return err
	}
	task, err := resolveCronTarget(cli, args[0])
	if err != nil {
		return err
	}
	if err := cli.DeleteCronTask(task.ID); err != nil {
		return err
	}
	fmt.Printf("Removed cron task %s (%s)\n", task.ID, displayOrDash(task.Name))
	return nil
}

func runCronEnableDisable(resolve func() (*client.Client, error), enable bool, args []string) error {
	action := "disable"
	if enable {
		action = "enable"
	}
	if len(args) != 1 {
		return fmt.Errorf("cron %s requires exactly 1 argument <name-or-id>", action)
	}
	cli, err := resolve()
	if err != nil {
		return err
	}
	task, err := resolveCronTarget(cli, args[0])
	if err != nil {
		return err
	}
	var st *client.CronTaskStatus
	if enable {
		st, err = cli.EnableCronTask(task.ID)
	} else {
		st, err = cli.DisableCronTask(task.ID)
	}
	if err != nil {
		return err
	}
	label := action
	if len(label) > 0 {
		label = strings.ToUpper(label[:1]) + label[1:]
	}
	fmt.Printf("%sd cron task %s (%s)\n", label, st.ID, displayOrDash(st.Name))
	return nil
}

func runCronRun(resolve func() (*client.Client, error), args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("cron run requires exactly 1 argument <name-or-id>")
	}
	cli, err := resolve()
	if err != nil {
		return err
	}
	task, err := resolveCronTarget(cli, args[0])
	if err != nil {
		return err
	}
	st, err := cli.RunCronTask(task.ID)
	if err != nil {
		return err
	}
	fmt.Printf("Triggered cron task %s (%s) status=%s\n", st.ID, displayOrDash(st.Name), displayOrDash(st.Status))
	return nil
}

func runCronLogs(resolve func() (*client.Client, error), args []string) error {
	lines := 100
	args, err := flags.
		Int("--lines", &lines).
		Help("-h,--help", cronLogsHelp).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) != 1 {
		return fmt.Errorf("cron logs requires exactly 1 argument <name-or-id>")
	}
	if lines <= 0 {
		return fmt.Errorf("--lines must be greater than 0")
	}
	cli, err := resolve()
	if err != nil {
		return err
	}
	task, err := resolveCronTarget(cli, args[0])
	if err != nil {
		return err
	}
	if strings.TrimSpace(task.LogPath) == "" {
		return fmt.Errorf("cron task %s has no log path", task.ID)
	}
	fmt.Printf("Streaming logs for %s (%s)\n", displayOrDash(task.Name), task.ID)
	fmt.Printf("Log path: %s\n", task.LogPath)
	fmt.Println("Press Ctrl+C to stop.")
	return cli.StreamLogFile(task.LogPath, lines, func(ev client.LogStreamEvent) {
		switch ev.Type {
		case "log":
			if ev.Message != "" {
				fmt.Println(ev.Message)
			}
		case "status":
			if ev.Message != "" {
				fmt.Println(ev.Message)
			} else if ev.Status != "" {
				fmt.Println(ev.Status)
			}
		}
	})
}

func runCronHistory(resolve func() (*client.Client, error), args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("cron history requires exactly 1 argument <name-or-id>")
	}
	cli, err := resolve()
	if err != nil {
		return err
	}
	task, err := resolveCronTarget(cli, args[0])
	if err != nil {
		return err
	}
	runs, err := cli.CronTaskHistory(task.ID)
	if err != nil {
		return err
	}
	if len(runs) == 0 {
		fmt.Println("No recent runs.")
		return nil
	}
	for i, r := range runs {
		if i > 0 {
			fmt.Println()
		}
		fmt.Printf("Started:  %s\n", formatAgentTime(r.StartedAt))
		if r.FinishedAt != "" {
			fmt.Printf("Finished: %s\n", formatAgentTime(r.FinishedAt))
		}
		if r.ExitCode != nil {
			fmt.Printf("Exit:     %d\n", *r.ExitCode)
		}
		if r.Error != "" {
			fmt.Printf("Error:    %s\n", r.Error)
		}
	}
	return nil
}

func resolveCronTarget(cli *client.Client, idOrName string) (*client.CronTaskStatus, error) {
	tasks, err := cli.ListCronTasks()
	if err != nil {
		return nil, err
	}
	return client.FindCronTask(tasks, idOrName)
}

func printCronTask(task client.CronTaskStatus) {
	const labelWidth = 14
	label := func(name string) string {
		return fmt.Sprintf("  %-*s", labelWidth, name+":")
	}
	fmt.Printf("%s %s\n", label("Name"), displayOrDash(task.Name))
	fmt.Printf("%s %s\n", label("ID"), task.ID)
	fmt.Printf("%s %s\n", label("Status"), displayOrDash(task.Status))
	fmt.Printf("%s %s\n", label("Enabled"), boolWord(task.Enabled))
	fmt.Printf("%s %s\n", label("Schedule"), formatCronSchedule(task))
	fmt.Printf("%s %s\n", label("Timeout"), displayOrDash(task.Timeout))
	fmt.Printf("%s %s\n", label("Command"), displayOrDash(task.Command))
	fmt.Printf("%s %s\n", label("Work Dir"), displayOrDash(task.WorkingDir))
	fmt.Printf("%s %s\n", label("Log Path"), displayOrDash(task.LogPath))
	if task.PID > 0 {
		fmt.Printf("%s %s\n", label("PID"), formatOptionalInt(task.PID))
	}
	if task.LastStartedAt != "" {
		fmt.Printf("%s %s\n", label("Last Start"), formatAgentTime(task.LastStartedAt))
	}
	if task.LastFinishedAt != "" {
		fmt.Printf("%s %s\n", label("Last Finish"), formatAgentTime(task.LastFinishedAt))
	}
	if task.NextRunAt != "" {
		fmt.Printf("%s %s\n", label("Next Run"), formatAgentTime(task.NextRunAt))
	}
	if task.LastError != "" {
		fmt.Printf("%s %s\n", label("Last Error"), task.LastError)
	}
}

func formatCronSchedule(task client.CronTaskStatus) string {
	switch task.ScheduleMode {
	case "interval":
		return "every " + displayOrDash(task.Interval)
	case "cron":
		return "cron(UTC) " + displayOrDash(task.CronExpr)
	default:
		return displayOrDash(task.ScheduleMode)
	}
}

// resolveCronScheduleFlags picks exactly one schedule source.
func resolveCronScheduleFlags(every, cronLocal, cronUTC string) (mode, interval, cronExpr string, err error) {
	n := 0
	if strings.TrimSpace(every) != "" {
		n++
	}
	if strings.TrimSpace(cronLocal) != "" {
		n++
	}
	if strings.TrimSpace(cronUTC) != "" {
		n++
	}
	if n == 0 {
		return "", "", "", fmt.Errorf("exactly one of --every, --cron, or --cron-utc is required")
	}
	if n > 1 {
		return "", "", "", fmt.Errorf("--every, --cron, and --cron-utc are mutually exclusive")
	}
	if every != "" {
		if _, e := time.ParseDuration(every); e != nil {
			return "", "", "", fmt.Errorf("invalid --every duration: %w", e)
		}
		return "interval", every, "", nil
	}
	if cronUTC != "" {
		if err := validateFiveFieldCron(cronUTC); err != nil {
			return "", "", "", fmt.Errorf("invalid --cron-utc: %w", err)
		}
		return "cron", "", strings.TrimSpace(cronUTC), nil
	}
	// --cron local convert
	utcExpr, err := convertLocalCronToUTC(strings.TrimSpace(cronLocal), time.Local)
	if err != nil {
		return "", "", "", err
	}
	return "cron", "", utcExpr, nil
}

func validateFiveFieldCron(expr string) error {
	fields := strings.Fields(strings.TrimSpace(expr))
	if len(fields) != 5 {
		return fmt.Errorf("want 5 fields, got %d", len(fields))
	}
	return nil
}

// convertLocalCronToUTC converts a simple local 5-field cron to UTC when safe.
// Unsafe patterns (ranges, lists, steps, DST zones with non-fixed offset) error
// and mention --cron-utc.
func convertLocalCronToUTC(expr string, loc *time.Location) (string, error) {
	fields := strings.Fields(strings.TrimSpace(expr))
	if len(fields) != 5 {
		return "", fmt.Errorf("invalid --cron expression (want 5 fields); use --cron-utc for explicit UTC")
	}
	for i, f := range fields {
		if !isSimpleCronToken(f) {
			return "", fmt.Errorf("unsafe --cron pattern %q (field %d): cannot auto-convert ranges/lists/steps; use --cron-utc", f, i+1)
		}
	}
	// Fixed offset required (no DST ambiguity).
	if !zoneFixedOffset(loc) {
		return "", fmt.Errorf("local timezone has DST or variable offset; cannot safely convert --cron; use --cron-utc")
	}

	minStr, hourStr := fields[0], fields[1]
	// Minute and hour must be pure numbers for conversion (not *).
	if minStr == "*" || hourStr == "*" {
		return "", fmt.Errorf("unsafe --cron: minute/hour wildcards need manual UTC conversion; use --cron-utc")
	}
	min, err := strconv.Atoi(minStr)
	if err != nil {
		return "", fmt.Errorf("invalid minute in --cron; use --cron-utc")
	}
	hour, err := strconv.Atoi(hourStr)
	if err != nil {
		return "", fmt.Errorf("invalid hour in --cron; use --cron-utc")
	}

	// Build a local wall time and convert.
	now := time.Now().In(loc)
	localT := time.Date(now.Year(), now.Month(), now.Day(), hour, min, 0, 0, loc)
	utcT := localT.UTC()

	// If day shifts and dom/month/dow are not all wildcards, refuse.
	if utcT.Day() != localT.Day() || utcT.Month() != localT.Month() || utcT.Year() != localT.Year() {
		if fields[2] != "*" || fields[3] != "*" || fields[4] != "*" {
			return "", fmt.Errorf("unsafe --cron: conversion crosses day boundary with constrained date fields; use --cron-utc")
		}
	}

	// Keep dom/month/dow as provided when all '*'; if DOW is numeric and day shifted, unsafe already handled.
	return fmt.Sprintf("%d %d %s %s %s", utcT.Minute(), utcT.Hour(), fields[2], fields[3], fields[4]), nil
}

func isSimpleCronToken(f string) bool {
	if f == "*" {
		return true
	}
	// pure integer only — no '-', ',', '/'
	if strings.ContainsAny(f, "-,/") {
		return false
	}
	_, err := strconv.Atoi(f)
	return err == nil
}

func zoneFixedOffset(loc *time.Location) bool {
	if loc == nil {
		return false
	}
	jan := time.Date(2024, 1, 15, 12, 0, 0, 0, loc)
	jul := time.Date(2024, 7, 15, 12, 0, 0, 0, loc)
	_, off1 := jan.Zone()
	_, off2 := jul.Zone()
	return off1 == off2
}
