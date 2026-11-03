package portforward

import (
	"fmt"
	"strconv"
)

// Caller holds m.mu. Bare ports are accepted only when they identify one mapping.
func (m *Manager) mappingIDLocked(selector string) (string, error) {
	if _, ok := m.tunnels[selector]; ok {
		return selector, nil
	}
	port, err := strconv.Atoi(selector)
	if err != nil {
		return "", fmt.Errorf("mapping %q not being forwarded", selector)
	}
	id := ""
	for key, t := range m.tunnels {
		if t.port == port {
			if id != "" {
				return "", fmt.Errorf("port %d matches multiple mappings; specify a mapping ID", port)
			}
			id = key
		}
	}
	if id == "" {
		return "", fmt.Errorf("port %d is not being forwarded", port)
	}
	return id, nil
}
func (m *Manager) GetLogsSelector(selector string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, err := m.mappingIDLocked(selector)
	if err != nil {
		return nil, err
	}
	t := m.tunnels[id]
	if t.logs == nil {
		return []string{}, nil
	}
	return t.logs.Lines(), nil
}
