package sshcmd

import (
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func TestLocalRelayUnixSplicesAndRemovesSocket(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "ra-ssh-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	socket := filepath.Join(dir, "relay.sock")
	relay := &LocalRelay{
		Network: "unix",
		Address: socket,
		Dial: func() (net.Conn, error) {
			client, server := net.Pipe()
			go func() {
				defer server.Close()
				_, _ = io.Copy(server, server)
			}()
			return client, nil
		},
	}
	if err := relay.Start(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(socket)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("socket mode = %o, want 0600", info.Mode().Perm())
	}
	conn, err := net.Dial("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := io.WriteString(conn, "relay-ok"); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, len("relay-ok"))
	if _, err := io.ReadFull(conn, buf); err != nil {
		t.Fatal(err)
	}
	if got := string(buf); got != "relay-ok" {
		t.Fatalf("relay response = %q", got)
	}
	if err := relay.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(socket); !os.IsNotExist(err) {
		t.Fatalf("socket remains after Close: %v", err)
	}
}

func TestConfigLocalInstallAndUninstall(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".ai-critic", "ssh")
	configPath := filepath.Join(home, ".ssh", "config")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("Host github\n  HostName github.com\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var output strings.Builder
	if err := RunConfigLocal([]string{"install", "--host", "go-cache"}, home, configDir, &output, io.Discard); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.HasPrefix(text, configLocalBegin+"\nInclude ") || !strings.Contains(text, "Host github") {
		t.Fatalf("unexpected config:\n%s", text)
	}
	profile, err := LoadConfigLocalProfile(configDir)
	if err != nil {
		t.Fatal(err)
	}
	if profile.Host != "go-cache" || profile.User != "agent" {
		t.Fatalf("profile = %#v", profile)
	}
	var status, warnings strings.Builder
	if err := RunConfigLocal([]string{"status"}, home, configDir, &status, &warnings); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(status.String(), "installed:  yes") || !strings.Contains(status.String(), "session:    inactive") {
		t.Fatalf("unexpected status:\n%s", status.String())
	}
	if !strings.Contains(warnings.String(), "warning: start the relay") {
		t.Fatalf("missing stopped-relay warning: %q", warnings.String())
	}
	if err := RunConfigLocal([]string{"uninstall"}, home, configDir, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), configLocalBegin) || !strings.Contains(string(data), "Host github") {
		t.Fatalf("unexpected config after uninstall:\n%s", data)
	}
	if err := RunConfigLocal([]string{"install", "--host", "github"}, home, configDir, io.Discard, io.Discard); err == nil {
		t.Fatal("install unexpectedly accepted an existing Host github entry")
	}
}

func TestCryptoSSHRunnerUsesUnixSocket(t *testing.T) {
	pair, err := GenerateClientKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	server := &AdhocServer{ForcePipeShell: true}
	server.SetAuthorizedKeys([]ssh.PublicKey{pair.Public})
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	dir, err := os.MkdirTemp("/tmp", "ra-ssh-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	socket := filepath.Join(dir, "relay.sock")
	relay := &LocalRelay{Network: "unix", Address: socket, Dial: DialTCP(server.Addr())}
	if err := relay.Start(); err != nil {
		t.Fatal(err)
	}
	defer relay.Close()
	var output strings.Builder
	runner := &CryptoSSHRunner{
		Signer:                pair.Signer,
		Stdout:                &output,
		InsecureIgnoreHostKey: true,
		ForceCrypto:           true,
	}
	if err := runner.Run(&Session{LocalSocket: socket, User: "agent", Alive: true}, []string{"printf", "unix-ok"}, RunnerOpts{}); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); got != "unix-ok" {
		t.Fatalf("command output = %q", got)
	}
}

func TestAdhocServerSFTP(t *testing.T) {
	pair, err := GenerateClientKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	server := &AdhocServer{}
	server.SetAuthorizedKeys([]ssh.PublicKey{pair.Public})
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	client, err := ssh.Dial("tcp", server.Addr(), &ssh.ClientConfig{
		User:            "agent",
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(pair.Signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		t.Fatal(err)
	}
	defer sftpClient.Close()
	path := filepath.Join(t.TempDir(), "sftp-smoke.txt")
	file, err := sftpClient.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("sftp-ok")); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := sftpClient.Remove(path); err != nil {
		t.Fatal(err)
	}
}
