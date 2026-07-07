package wsproxy_singbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/xhd2015/ai-critic/client"
)

const BrewInstallSingBoxCmd = "brew install sing-box"

func PrintCommand(cmd string) {
	fmt.Printf("$ %s\n", cmd)
}

func SingBoxRunCommand(sudo bool, configPath string) string {
	if sudo {
		return fmt.Sprintf("sudo sing-box run -c %s", shellQuote(configPath))
	}
	return fmt.Sprintf("sing-box run -c %s", shellQuote(configPath))
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '/' && r != '_' && r != '-' && r != '.' && r != ':' {
			return fmt.Sprintf("%q", s)
		}
	}
	return s
}

type VMessParams struct {
	VMessLink string `json:"vmess_link"`
	Host      string `json:"host"`
	Port      string `json:"port"`
	UUID      string `json:"uuid"`
	AlterID   string `json:"alter_id"`
	Network   string `json:"network"`
	Type      string `json:"type"`
	Path      string `json:"path"`
	TLS       string `json:"tls"`
}

type ClientConfigOptions struct {
	OutputFile string
}

type RunTunOptions struct {
	ConfigFile  string
	Yes         bool
	NoInstall   bool
	NoSetupSudo bool
	Detach      bool
	HttpOnly    bool
	Policy      *DomainPolicy
	DNSHijack   bool
}

// RunHttpOnlyOptions is deprecated; use RunTunOptions with HttpOnly set.
type RunHttpOnlyOptions = RunTunOptions

type BuildConfigOptions struct {
	BindInterface   string
	LocalSocksPort  int // when > 0, proxy outbound is SOCKS to local xray sidecar
	HttpOnly        bool
	Policy          *DomainPolicy
	DNSHijack       bool // http-only: optional; full VPN always hijacks DNS
	InitialUseProxy bool // http-only selector default when ws-proxy is up
}

type TestHooks struct {
	LookPath      func(name string) (string, error)
	IsTTY         func() bool
	Confirm       func(prompt string) bool
	BrewInstall   func() error
	Geteuid       func() int
	RunSingBox    func(ctx context.Context, sudo bool, configPath string) error
	StartDetached   func(configPath, logPath string, useSudo bool) (pid int, err error)
	EnsureSudoSetup func(singBoxPath string, noSetup bool) error
	FetchVMess      func(c *client.Client) (*VMessParams, error)
	UserCacheDir     func() (string, error)
	StartXraySidecar func(ctx context.Context, vmess *VMessParams) (*XraySidecar, error)
}

var currentHooks = TestHooks{
	LookPath: defaultLookPath,
	IsTTY:    defaultIsTTY,
	Confirm: func(prompt string) bool {
		fmt.Print(prompt)
		var s string
		fmt.Scanln(&s)
		return s == "y" || s == "Y" || s == "yes"
	},
	BrewInstall: func() error {
		cmd := exec.Command("brew", "install", "sing-box")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	},
	Geteuid: os.Geteuid,
	RunSingBox: runSingBoxForeground,
	StartDetached: func(configPath, logPath string, useSudo bool) (int, error) {
		PrintCommand(SingBoxRunCommand(useSudo, configPath))
		args := []string{"sing-box", "run", "-c", configPath}
		if useSudo {
			args = append([]string{"sudo"}, args...)
		}
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Env = singBoxProcessEnv()
		logFile, err := os.Create(logPath)
		if err != nil {
			return 0, fmt.Errorf("create log file: %w", err)
		}
		cmd.Stdout = logFile
		cmd.Stderr = logFile
		if err := cmd.Start(); err != nil {
			logFile.Close()
			return 0, err
		}
		return cmd.Process.Pid, nil
	},
	FetchVMess: nil,
	UserCacheDir: func() (string, error) {
		return os.UserCacheDir()
	},
	EnsureSudoSetup: defaultEnsureSudoSetup,
}

func defaultLookPath(name string) (string, error) {
	return exec.LookPath(name)
}

func defaultIsTTY() bool {
	fi, _ := os.Stdout.Stat()
	return fi != nil && fi.Mode()&os.ModeCharDevice != 0
}

func InstallTestHooks(h TestHooks) func() {
	old := currentHooks
	currentHooks = h
	return func() {
		currentHooks = old
	}
}
