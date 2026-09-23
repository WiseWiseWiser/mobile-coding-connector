package services

// Preset is a predefined template for a user service. Selecting one prefills
// the Add form; nothing is created until the user saves it.
type Preset struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Command     string `json:"command"`
	// Port prefills the port-forward field. Zero means the preset needs none.
	Port int `json:"port,omitempty"`
}

// Presets lists the command templates this server ships. They cover the
// binaries built from cmd/, which are the only helpers aimed at being run as a
// managed service.
func Presets() []Preset {
	return []Preset{
		{
			ID:          "forward-proxy",
			Name:        "Forward Proxy",
			Description: "HTTP/HTTPS forward proxy that chains through an upstream proxy",
			Command:     "forward-proxy --listen :8888 --upstream-proxy http://127.0.0.1:7890",
			Port:        8888,
		},
		{
			ID:          "basic-auth-proxy",
			Name:        "Basic Auth Proxy",
			Description: "Cookie-based Basic Auth gate in front of a local port",
			Command:     "basic-auth-proxy --port 8080 --backend-port 3000",
			Port:        8080,
		},
	}
}
