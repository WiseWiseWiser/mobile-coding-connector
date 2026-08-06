package agentcli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh"

	"github.com/xhd2015/ai-critic/client"
	"github.com/xhd2015/ai-critic/cmd/agentcli/sshcmd"
	"github.com/xhd2015/ai-critic/cmd/agentcli/testhooks"
)

// runSSH dispatches the `ssh` subcommand with FileSessionStore and CryptoSSHRunner.
// Session root lives under ~/.ai-critic (or testhooks home override).
// --serve wires Dial via BuildSSHTunnelDial when a Client is resolvable.
// Client identity (CryptoSSHRunner.Signer) comes from EnsureClientKeyPair under configDir.
func runSSH(args []string, resolve func() (*client.Client, error)) error {
	home, err := testhooks.UserHomeDir()
	if err != nil {
		return err
	}
	root := filepath.Join(home, ".ai-critic")
	store := &sshcmd.FileSessionStore{Root: root}
	configDir := filepath.Join(root, "ssh")

	kp, err := sshcmd.EnsureClientKeyPair(configDir)
	if err != nil {
		return fmt.Errorf("ensure client key pair: %w", err)
	}

	runner := &sshcmd.CryptoSSHRunner{
		Signer:                kp.Signer,
		Stdout:                os.Stdout,
		Stderr:                os.Stderr,
		Stdin:                 os.Stdin,
		InsecureIgnoreHostKey: true,
	}

	var cli *client.Client
	if resolve != nil {
		// Best-effort: missing config leaves Dial nil (P3 / offline serve error path).
		if c, err := resolve(); err == nil {
			cli = c
		}
	}

	serve := NewSSHServeStarter(store, configDir, cli)

	return sshcmd.Run(sshcmd.Options{
		Args:      args,
		ProfileID: "default",
		Store:     store,
		Serve:     serve,
		Runner:    runner,
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
	})
}

// BuildSSHTunnelDial creates a remote SSH session and returns a DialFunc that
// opens a new duplex WS tunnel on each call. Exported for L2 unit tests.
func BuildSSHTunnelDial(c *client.Client, publicKey string) (sshcmd.DialFunc, *client.SSHSessionInfo, error) {
	if c == nil {
		return nil, nil, errors.New("client is nil")
	}
	info, err := c.CreateSSHSession(client.CreateSSHSessionRequest{PublicKey: publicKey})
	if err != nil {
		return nil, nil, err
	}
	return c.SSHTunnelDialFunc(info.SessionID), info, nil
}

// sshServeStarter adapts ServeService to sshcmd.ServeStarter (Start with ServeOpts).
// When Client is non-nil, BuildSSHTunnelDial supplies ServeService.Dial.
type sshServeStarter struct {
	store     *sshcmd.FileSessionStore
	configDir string
	client    *client.Client
	// PublicKeyOpenSSH overrides key material when set (tests).
	PublicKeyOpenSSH string
}

// NewSSHServeStarter constructs a serve starter. client may be nil (Dial stays nil).
func NewSSHServeStarter(store *sshcmd.FileSessionStore, configDir string, cli *client.Client) *sshServeStarter {
	return &sshServeStarter{
		store:     store,
		configDir: configDir,
		client:    cli,
	}
}

func (s *sshServeStarter) Start(opts sshcmd.ServeOpts) error {
	if s == nil {
		return errors.New("serve starter not configured")
	}
	profileID := opts.ProfileID
	if profileID == "" {
		profileID = "default"
	}

	var dial sshcmd.DialFunc
	if s.client != nil {
		pub := s.PublicKeyOpenSSH
		if pub == "" {
			kp, err := sshcmd.EnsureClientKeyPair(s.configDir)
			if err != nil {
				return err
			}
			pub = strings.TrimSpace(string(ssh.MarshalAuthorizedKey(kp.Public)))
		}
		d, _, err := BuildSSHTunnelDial(s.client, pub)
		if err != nil {
			return fmt.Errorf("build ssh tunnel dial: %w", err)
		}
		dial = d
	}

	svc := &sshcmd.ServeService{
		Store:     s.store,
		ProfileID: profileID,
		Dial:      dial,
		User:      "agent",
		ConfigDir: s.configDir,
		ServePID:  os.Getpid(),
		Stdout:    opts.Stdout,
		Quiet:     opts.Quiet,
	}
	if svc.Stdout == nil {
		svc.Stdout = os.Stdout
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return svc.Start(ctx)
}
