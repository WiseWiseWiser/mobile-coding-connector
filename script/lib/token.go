package lib

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
)

const CookieName = "ai-critic-token"

var CredentialsFile = config.CredentialsFile

func LoadFirstToken() (string, error) {
	f, err := os.Open(CredentialsFile)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			return line, nil
		}
	}
	return "", scanner.Err()
}

func LoadFirstTokenFromHome() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	credFile := filepath.Join(homeDir, ".ai-critic", "server-credentials")
	f, err := os.Open(credFile)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			return line, nil
		}
	}
	return "", scanner.Err()
}
