package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/xhd2015/less-gen/flags"
)

const localReapHelp = `Usage: remote-agent local reap [options]

List defunct (zombie) processes on the local machine, and optionally
nudge their parents to reap them.

A zombie process is a child whose parent has not called wait(). Only
the parent can reap it; an external tool can at best:
  - send SIGCHLD to the parent, to prompt its signal handler to reap
  - kill the parent, so init adopts and reaps the zombies

Options:
  --signal                 Send SIGCHLD to each unique parent of a zombie.
  --kill-parent            Send SIGTERM to each unique parent of a zombie.
                           Use with caution; the parent process will exit.
  --filter NAME_SUBSTR     Only list zombies whose command name contains
                           NAME_SUBSTR (e.g. 'ai-critic').
  -h, --help               Show this help message.
`

type zombieInfo struct {
	PID      int
	PPID     int
	Name     string
	ParentOK bool
	ParentPS string
}

func runLocalReap(args []string) error {
	var doSignal bool
	var doKillParent bool
	var filter string

	args, err := flags.
		Bool("--signal", &doSignal).
		Bool("--kill-parent", &doKillParent).
		String("--filter", &filter).
		Help("-h,--help", localReapHelp).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) > 0 {
		return fmt.Errorf("local reap takes no positional arguments, got %v", args)
	}

	zombies, err := listZombies(filter)
	if err != nil {
		return err
	}

	if len(zombies) == 0 {
		fmt.Println("No defunct processes found.")
		return nil
	}

	fmt.Printf("Found %d defunct process(es):\n", len(zombies))
	fmt.Printf("  %-8s %-8s %s\n", "PID", "PPID", "NAME")
	for _, z := range zombies {
		fmt.Printf("  %-8d %-8d %s\n", z.PID, z.PPID, z.Name)
	}

	if !doSignal && !doKillParent {
		fmt.Println("\nTip: pass --signal to send SIGCHLD to parents (nudging them to reap),")
		fmt.Println("     or --kill-parent to SIGTERM the parents so init adopts & reaps them.")
		return nil
	}

	parents := uniqueParents(zombies)
	sig := syscall.SIGCHLD
	action := "SIGCHLD"
	if doKillParent {
		sig = syscall.SIGTERM
		action = "SIGTERM"
	}

	fmt.Printf("\nSending %s to %d unique parent(s):\n", action, len(parents))
	for _, ppid := range parents {
		pname, _ := readProcComm(ppid)
		if err := syscall.Kill(ppid, sig); err != nil {
			fmt.Printf("  pid=%d (%s): %v\n", ppid, pname, err)
			continue
		}
		fmt.Printf("  pid=%d (%s): %s sent\n", ppid, pname, action)
	}
	return nil
}

// listZombies scans /proc for processes in state 'Z' (zombie). If
// filter is non-empty, only zombies whose command name contains the
// substring are returned.
func listZombies(filter string) ([]zombieInfo, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("read /proc: %w (local reap is only supported on Linux)", err)
	}

	var zombies []zombieInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		state, ppid, name, ok := readProcStat(pid)
		if !ok {
			continue
		}
		if state != "Z" {
			continue
		}
		if filter != "" && !strings.Contains(name, filter) {
			continue
		}
		zombies = append(zombies, zombieInfo{
			PID:  pid,
			PPID: ppid,
			Name: name,
		})
	}

	sort.Slice(zombies, func(i, j int) bool {
		if zombies[i].PPID != zombies[j].PPID {
			return zombies[i].PPID < zombies[j].PPID
		}
		return zombies[i].PID < zombies[j].PID
	})
	return zombies, nil
}

// readProcStat parses /proc/<pid>/stat and returns (state, ppid, comm, ok).
// The format is: pid (comm) state ppid ...
// comm may contain spaces and parentheses, so we split on the last ')'.
func readProcStat(pid int) (string, int, string, bool) {
	data, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
	if err != nil {
		return "", 0, "", false
	}
	s := string(data)
	rparen := strings.LastIndex(s, ")")
	if rparen < 0 {
		return "", 0, "", false
	}
	lparen := strings.Index(s, "(")
	if lparen < 0 || lparen >= rparen {
		return "", 0, "", false
	}
	comm := s[lparen+1 : rparen]
	rest := strings.Fields(s[rparen+1:])
	if len(rest) < 2 {
		return "", 0, "", false
	}
	state := rest[0]
	ppid, err := strconv.Atoi(rest[1])
	if err != nil {
		return "", 0, "", false
	}
	return state, ppid, comm, true
}

func readProcComm(pid int) (string, bool) {
	data, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "comm"))
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(data)), true
}

func uniqueParents(zs []zombieInfo) []int {
	seen := map[int]bool{}
	var out []int
	for _, z := range zs {
		if z.PPID <= 1 || seen[z.PPID] {
			continue
		}
		seen[z.PPID] = true
		out = append(out, z.PPID)
	}
	sort.Ints(out)
	return out
}
