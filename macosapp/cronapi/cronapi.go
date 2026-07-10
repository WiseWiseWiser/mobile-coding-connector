// Package cronapi provides pure request builders for menu-bar cron task control
// against a remote (or local) AI Critic server. Swift ServiceClient / ServerClient
// mirror these paths.
package cronapi

import (
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

// ListCronTasksPath is GET path for listing cron tasks (no all=1 query).
func ListCronTasksPath() string {
	return "/api/cron-tasks"
}

// CronAction is a POST control action on a cron task.
type CronAction string

const (
	ActionRun     CronAction = "run"
	ActionEnable  CronAction = "enable"
	ActionDisable CronAction = "disable"
)

// CronActionPath builds POST path: /api/cron-tasks/{action}?id={id}
func CronActionPath(action CronAction, id string) string {
	q := url.Values{}
	q.Set("id", id)
	return fmt.Sprintf("/api/cron-tasks/%s?%s", action, q.Encode())
}

// CronRequest is a fully resolved HTTP request plan for the cron client.
type CronRequest struct {
	Method  string
	URL     string
	Headers map[string]string
}

// NormalizeBaseURL trims trailing slashes from a server base URL.
func NormalizeBaseURL(base string) string {
	return strings.TrimRight(strings.TrimSpace(base), "/")
}

// BuildListCronTasksRequest returns GET list-cron-tasks request with optional Bearer auth.
func BuildListCronTasksRequest(baseURL, token string) (CronRequest, error) {
	return build(baseURL, "GET", ListCronTasksPath(), token)
}

// BuildCronActionRequest returns POST cron-action request with optional Bearer auth.
func BuildCronActionRequest(baseURL, token string, action CronAction, id string) (CronRequest, error) {
	if strings.TrimSpace(id) == "" {
		return CronRequest{}, fmt.Errorf("task id is required")
	}
	switch action {
	case ActionRun, ActionEnable, ActionDisable:
	default:
		return CronRequest{}, fmt.Errorf("unknown cron action %q", action)
	}
	return build(baseURL, "POST", CronActionPath(action, id), token)
}

func build(baseURL, method, path string, token string) (CronRequest, error) {
	base := NormalizeBaseURL(baseURL)
	if base == "" {
		return CronRequest{}, fmt.Errorf("base URL is required")
	}
	if !strings.Contains(base, "://") {
		return CronRequest{}, fmt.Errorf("base URL must include scheme")
	}
	headers := map[string]string{}
	if h := AuthorizationHeader(token); h != "" {
		headers["Authorization"] = h
	}
	return CronRequest{
		Method:  method,
		URL:     base + path,
		Headers: headers,
	}, nil
}
