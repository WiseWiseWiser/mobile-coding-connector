package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	"golang.org/x/crypto/ssh"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
)

const keyBits = 3072

var (
	privateKeyFile = config.EncKeyFile
	publicKeyFile  = config.EncKeyPubFile
)

var help = fmt.Sprintf(`
Usage: go run ./script/crypto/gen

Generates an RSA key pair for encrypting SSH private keys in transit.

Files generated:
  %s      - RSA private key (OpenSSH format, used by server for decryption)
  %s  - RSA public key (OpenSSH format)

The server reads these files to provide RSA-OAEP encryption for SSH keys
sent from the frontend. If these files don't exist, the server will not
provide an encryption public key, and the frontend will refuse to send
SSH private keys to the server.
`, config.EncKeyFile, config.EncKeyPubFile)

func main() {
	err := Handle(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func Handle(args []string) error {
	_, err := flags.Help("-h,--help", help).Parse(args)
	if err != nil {
		return err
	}

	// Check if files already exist
	if _, err := os.Stat(privateKeyFile); err == nil {
		fmt.Printf("Key pair already exists: %s\n", privateKeyFile)
		fmt.Println("Delete the existing files to regenerate.")
		return nil
	}

	fmt.Printf("Generating %d-bit RSA key pair...\n", keyBits)

	// Generate RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, keyBits)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key: %v", err)
	}

	// Write private key in OpenSSH format
	privBlock, err := ssh.MarshalPrivateKey(privateKey, "")
	if err != nil {
		return fmt.Errorf("failed to marshal private key to OpenSSH format: %v", err)
	}
	privPEM := pem.EncodeToMemory(privBlock)
	if err := os.WriteFile(privateKeyFile, privPEM, 0600); err != nil {
		return fmt.Errorf("failed to write private key file: %v", err)
	}
	fmt.Printf("Written: %s\n", privateKeyFile)

	// Write public key in OpenSSH format
	pubKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to create SSH public key: %v", err)
	}
	pubBytes := ssh.MarshalAuthorizedKey(pubKey)
	if err := os.WriteFile(publicKeyFile, pubBytes, 0644); err != nil {
		return fmt.Errorf("failed to write public key file: %v", err)
	}
	fmt.Printf("Written: %s\n", publicKeyFile)

	// Also show the SPKI PEM format for reference
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err == nil {
		pubPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: pubKeyBytes,
		})
		fmt.Println("\nSPKI PEM public key (used by frontend Web Crypto API):")
		fmt.Println(string(pubPEM))
	}

	fmt.Println("Done! The server will now provide the public key to the frontend for SSH key encryption.")
	return nil
}
