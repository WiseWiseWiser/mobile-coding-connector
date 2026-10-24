package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// PortMappingDesiredFile is the set of mappings to restore across server restarts.
// Presence of a port means it should stay live; names/auth files are separate.
var PortMappingDesiredFile = DataDir + "/port-mapping-desired.json"

// PortMappingDesiredSpec is the live-restore record for one port.
type PortMappingDesiredSpec struct {
	Hostname string `json:"hostname"`
	Provider string `json:"provider,omitempty"`
}

// PortMappingDesired maps a port number (decimal string) to a desired live mapping.
type PortMappingDesired map[string]PortMappingDesiredSpec

func portMappingDesiredKey(port int) string {
	return fmt.Sprintf("%d", port)
}

// LoadPortMappingDesired loads the desired-live mapping set.
func LoadPortMappingDesired() (PortMappingDesired, error) {
	data, err := os.ReadFile(PortMappingDesiredFile)
	if err != nil {
		if os.IsNotExist(err) {
			return make(PortMappingDesired), nil
		}
		return nil, fmt.Errorf("failed to read port mapping desired file: %w", err)
	}
	var mappings PortMappingDesired
	if err := json.Unmarshal(data, &mappings); err != nil {
		return nil, fmt.Errorf("failed to parse port mapping desired file: %w", err)
	}
	if mappings == nil {
		mappings = make(PortMappingDesired)
	}
	return mappings, nil
}

func savePortMappingDesired(mappings PortMappingDesired) error {
	if err := os.MkdirAll(DataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}
	data, err := json.MarshalIndent(mappings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal port mapping desired file: %w", err)
	}
	if err := os.WriteFile(PortMappingDesiredFile, append(data, '\n'), 0600); err != nil {
		return fmt.Errorf("failed to write port mapping desired file: %w", err)
	}
	return nil
}

// SetPortMappingDesired records that a port should stay published.
func SetPortMappingDesired(port int, spec PortMappingDesiredSpec) error {
	spec.Hostname = strings.TrimSpace(spec.Hostname)
	spec.Provider = strings.TrimSpace(spec.Provider)
	if spec.Hostname == "" {
		return fmt.Errorf("desired mapping hostname is required")
	}
	mappings, err := LoadPortMappingDesired()
	if err != nil {
		return err
	}
	mappings[portMappingDesiredKey(port)] = spec
	return savePortMappingDesired(mappings)
}

// DeletePortMappingDesired stops restoring a port (names/auth are untouched).
func DeletePortMappingDesired(port int) error {
	mappings, err := LoadPortMappingDesired()
	if err != nil {
		return err
	}
	key := portMappingDesiredKey(port)
	if _, ok := mappings[key]; !ok {
		return nil
	}
	delete(mappings, key)
	return savePortMappingDesired(mappings)
}
