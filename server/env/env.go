package env

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const (
	dotEnvFile      = ".env"
	dotEnvLocalFile = ".env.local"
)

// Load reads environment variables from .env then .env.local.
// Values from .env.local override .env.
func Load() error {
	if err := loadFile(dotEnvFile); err != nil {
		return err
	}
	if err := loadFile(dotEnvLocalFile); err != nil {
		return err
	}
	return nil
}

func loadFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
			if line == "" {
				continue
			}
		}

		key, val, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("invalid env format in %s:%d", path, lineNo)
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if key == "" {
			return fmt.Errorf("empty env key in %s:%d", path, lineNo)
		}

		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}

		if err := os.Setenv(key, val); err != nil {
			return fmt.Errorf("set env %s from %s:%d: %w", key, path, lineNo, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	return nil
}
