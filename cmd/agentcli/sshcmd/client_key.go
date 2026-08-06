package sshcmd

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

// EnsureClientKeyPair loads or creates an ed25519 client identity under configDir.
//
// Layout:
//
//	{configDir}/id_ed25519      # private, mode 0600, ssh.ParsePrivateKey-able
//	{configDir}/id_ed25519.pub  # optional OpenSSH authorized_keys line
//
// If the private file exists and parses, the same pair is returned (stable identity).
// If it exists but is corrupt/unreadable, an error is returned (fail-closed; no silent regen).
// If missing, a new key is generated, written with mode 0600, and returned.
// Creates configDir (0755) when needed.
func EnsureClientKeyPair(configDir string) (*ClientKeyPair, error) {
	if configDir == "" {
		return nil, errors.New("configDir is required")
	}

	privPath := filepath.Join(configDir, "id_ed25519")

	data, err := os.ReadFile(privPath)
	if err == nil {
		signer, err := ssh.ParsePrivateKey(data)
		if err != nil {
			return nil, fmt.Errorf("parse private key %s: %w", privPath, err)
		}
		return &ClientKeyPair{
			Signer: signer,
			Public: signer.PublicKey(),
		}, nil
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read private key %s: %w", privPath, err)
	}

	// Missing: generate and persist.
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir configDir: %w", err)
	}

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ed25519: %w", err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		return nil, fmt.Errorf("signer from key: %w", err)
	}

	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		return nil, fmt.Errorf("marshal private key: %w", err)
	}
	pemBytes := pem.EncodeToMemory(block)
	if pemBytes == nil {
		return nil, errors.New("marshal private key: empty PEM")
	}
	if err := os.WriteFile(privPath, pemBytes, 0o600); err != nil {
		return nil, fmt.Errorf("write private key: %w", err)
	}

	// Optional public key file (OpenSSH authorized_keys line).
	pubPath := privPath + ".pub"
	_ = os.WriteFile(pubPath, ssh.MarshalAuthorizedKey(signer.PublicKey()), 0o644)

	return &ClientKeyPair{
		Signer: signer,
		Public: signer.PublicKey(),
	}, nil
}
