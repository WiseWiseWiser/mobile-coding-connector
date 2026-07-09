// Package serviceapi provides pure request builders for menu-bar service control
// against a remote (or local) AI Critic server. Swift ServiceClient mirrors these paths.
package serviceapi

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// AuthorizationHeader formats a Bearer token header value.
// Empty token returns "" (caller should omit the header).
func AuthorizationHeader(token string) string {
	if token == "" {
		return ""
	}
	return "Bearer " + token
}

// ListServicesPath is GET path for listing all services (including disabled).
func ListServicesPath() string {
	return "/api/services?all=1"
}

// ServiceAction is a POST control action on a service.
type ServiceAction string

const (
	ActionStart   ServiceAction = "start"
	ActionStop    ServiceAction = "stop"
	ActionRestart ServiceAction = "restart"
	ActionEnable  ServiceAction = "enable"
	ActionDisable ServiceAction = "disable"
)

// ServiceActionPath builds POST path for a service action: /api/services/{action}?id={id}
func ServiceActionPath(action ServiceAction, id string) string {
	q := url.Values{}
	q.Set("id", id)
	return fmt.Sprintf("/api/services/%s?%s", action, q.Encode())
}

// ServiceRequest is a fully resolved HTTP request plan for the service client.
type ServiceRequest struct {
	Method  string
	URL     string
	Headers map[string]string
}

// NormalizeBaseURL trims trailing slashes from a server base URL.
func NormalizeBaseURL(base string) string {
	return strings.TrimRight(strings.TrimSpace(base), "/")
}

// BuildListServicesRequest returns GET list-services request with optional Bearer auth.
func BuildListServicesRequest(baseURL, token string) (ServiceRequest, error) {
	return build(baseURL, "GET", ListServicesPath(), token)
}

// BuildServiceActionRequest returns POST service-action request with optional Bearer auth.
func BuildServiceActionRequest(baseURL, token string, action ServiceAction, id string) (ServiceRequest, error) {
	if strings.TrimSpace(id) == "" {
		return ServiceRequest{}, fmt.Errorf("service id is required")
	}
	switch action {
	case ActionStart, ActionStop, ActionRestart, ActionEnable, ActionDisable:
	default:
		return ServiceRequest{}, fmt.Errorf("unknown service action %q", action)
	}
	return build(baseURL, "POST", ServiceActionPath(action, id), token)
}

func build(baseURL, method, path string, token string) (ServiceRequest, error) {
	base := NormalizeBaseURL(baseURL)
	if base == "" {
		return ServiceRequest{}, fmt.Errorf("base URL is required")
	}
	if !strings.Contains(base, "://") {
		return ServiceRequest{}, fmt.Errorf("base URL must include scheme")
	}
	headers := map[string]string{}
	if h := AuthorizationHeader(token); h != "" {
		headers["Authorization"] = h
	}
	return ServiceRequest{
		Method:  method,
		URL:     base + path,
		Headers: headers,
	}, nil
}

// AcceptServiceActionBody reports whether body is a successful service-action
// payload after HTTP 200. Mirrors Swift ServiceClient.decodeServiceActionBody:
// empty body, {status,message?,service?}, bare ServiceStatus, or {"status":"ok"}.
func AcceptServiceActionBody(body []byte) (message string, ok bool) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return "", true
	}
	var generic map[string]json.RawMessage
	if err := json.Unmarshal(body, &generic); err != nil {
		return "", false
	}
	// enable/disable: message present
	if raw, has := generic["message"]; has {
		var msg string
		_ = json.Unmarshal(raw, &msg)
		return msg, true
	}
	// stop/restart: {"status":"ok"} or start ServiceStatus (has id)
	if _, hasStatus := generic["status"]; hasStatus {
		return "", true
	}
	if _, hasID := generic["id"]; hasID {
		// bare ServiceStatus from start
		return "", true
	}
	return "", false
}
