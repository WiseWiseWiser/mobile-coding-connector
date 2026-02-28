package exposed_opencode

// WebServerStatus represents the status of the OpenCode web server.
type WebServerStatus struct {
	Running          bool             `json:"running"`
	Port             int              `json:"port"`
	Domain           string           `json:"domain"`
	PortMapped       bool             `json:"port_mapped"`
	TargetPreference TargetPreference `json:"target_preference,omitempty"`
	ExposedDomain    string           `json:"exposed_domain,omitempty"`
	ConfigPath       string           `json:"config_path"`
	AuthProxyRunning bool             `json:"auth_proxy_running"`
	AuthProxyFound   bool             `json:"auth_proxy_found"`
	AuthProxyPath    string           `json:"auth_proxy_path"`
	OpencodePort     int              `json:"opencode_port"`
}

// WebServerControlResponse represents the response from a control operation.
type WebServerControlResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Running bool   `json:"running"`
}

// MapDomainRequest represents a request to map the domain via Cloudflare.
type MapDomainRequest struct {
	Provider string `json:"provider,omitempty"` // Optional: "cloudflare_owned" or "cloudflare_tunnel"
}

// MapDomainResponse represents the response from a domain mapping operation.
type MapDomainResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	PublicURL string `json:"public_url,omitempty"`
}
