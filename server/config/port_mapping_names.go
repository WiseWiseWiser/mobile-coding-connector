package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// PortMappingNamesFile is the path to the port mapping names JSON file
const PortMappingNamesFile = DataDir + "/port-mapping-names.json"

// PortMappingNameEntry represents a single port's last used domain mapping
// The key is the port number (as string in JSON), and value is the full domain name
// Example: {"8080": "myapp.example.com", "3000": "api.example.com"}
type PortMappingNames map[string]string

// LoadPortMappingNames loads the port mapping names from the JSON file
func LoadPortMappingNames() (PortMappingNames, error) {
	data, err := os.ReadFile(PortMappingNamesFile)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, return empty map
			return make(PortMappingNames), nil
		}
		return nil, fmt.Errorf("failed to read port mapping names file: %w", err)
	}

	var mappingNames PortMappingNames
	if err := json.Unmarshal(data, &mappingNames); err != nil {
		return nil, fmt.Errorf("failed to parse port mapping names file: %w", err)
	}

	return mappingNames, nil
}

// SavePortMappingNames saves the port mapping names to the JSON file
func SavePortMappingNames(mappingNames PortMappingNames) error {
	// Ensure directory exists
	if err := os.MkdirAll(DataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	data, err := json.MarshalIndent(mappingNames, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal port mapping names: %w", err)
	}

	if err := os.WriteFile(PortMappingNamesFile, append(data, '\n'), 0644); err != nil {
		return fmt.Errorf("failed to write port mapping names file: %w", err)
	}

	return nil
}

// GetPortMappingName gets the saved domain for a specific port
func GetPortMappingName(port int) (string, error) {
	mappings, err := LoadPortMappingNames()
	if err != nil {
		return "", err
	}

	// Use the port number as the key
	key := fmt.Sprintf("%d", port)
	return mappings[key], nil
}

// SetPortMappingName saves the domain for a specific port
func SetPortMappingName(port int, domain string) error {
	mappings, err := LoadPortMappingNames()
	if err != nil {
		return err
	}

	// Use the port number as the key
	key := fmt.Sprintf("%d", port)
	mappings[key] = domain

	return SavePortMappingNames(mappings)
}

// DeletePortMappingName removes the saved domain for a specific port
func DeletePortMappingName(port int) error {
	mappings, err := LoadPortMappingNames()
	if err != nil {
		return err
	}

	// Use the port number as the key
	key := fmt.Sprintf("%d", port)
	delete(mappings, key)

	return SavePortMappingNames(mappings)
}
