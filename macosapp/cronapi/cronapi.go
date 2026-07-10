// Package cronapi provides pure request builders for menu-bar cron task control
// against a remote (or local) AI Critic server. Swift ServiceClient / ServerClient
// mirror these paths.
package cronapi

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
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

// CreateCronTasksPath is POST path for creating a cron task.
func CreateCronTasksPath() string {
	return "/api/cron-tasks"
}

// UpdateCronTasksPath is PUT path for updating a cron task.
func UpdateCronTasksPath() string {
	return "/api/cron-tasks"
}

// DeleteCronTaskPath builds DELETE path: /api/cron-tasks?id={id}
func DeleteCronTaskPath(id string) string {
	q := url.Values{}
	q.Set("id", id)
	return "/api/cron-tasks?" + q.Encode()
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

// CronTaskDef is the create/update JSON body (no extraEnv in menu-bar UI).
// cronExpr as stored/sent is always UTC.
type CronTaskDef struct {
	ID           string `json:"id,omitempty"`
	Name         string `json:"name,omitempty"`
	Command      string `json:"command,omitempty"`
	WorkingDir   string `json:"workingDir,omitempty"`
	ScheduleMode string `json:"scheduleMode,omitempty"`
	Interval     string `json:"interval,omitempty"`
	CronExpr     string `json:"cronExpr,omitempty"`
	Timeout      string `json:"timeout,omitempty"`
	Enabled      *bool  `json:"enabled,omitempty"`
}

// CronRequest is a fully resolved HTTP request plan for the cron client.
type CronRequest struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    []byte
}

// NormalizeBaseURL trims trailing slashes from a server base URL.
func NormalizeBaseURL(base string) string {
	return strings.TrimRight(strings.TrimSpace(base), "/")
}

// BuildListCronTasksRequest returns GET list-cron-tasks request with optional Bearer auth.
func BuildListCronTasksRequest(baseURL, token string) (CronRequest, error) {
	return build(baseURL, "GET", ListCronTasksPath(), token, nil)
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
	return build(baseURL, "POST", CronActionPath(action, id), token, nil)
}

// BuildCreateCronTaskRequest returns POST /api/cron-tasks with JSON definition body.
func BuildCreateCronTaskRequest(baseURL, token string, def CronTaskDef) (CronRequest, error) {
	body, err := json.Marshal(def)
	if err != nil {
		return CronRequest{}, err
	}
	return build(baseURL, "POST", CreateCronTasksPath(), token, body)
}

// BuildUpdateCronTaskRequest returns PUT /api/cron-tasks with JSON body including id.
func BuildUpdateCronTaskRequest(baseURL, token string, def CronTaskDef) (CronRequest, error) {
	if strings.TrimSpace(def.ID) == "" {
		return CronRequest{}, fmt.Errorf("task id is required")
	}
	body, err := json.Marshal(def)
	if err != nil {
		return CronRequest{}, err
	}
	return build(baseURL, "PUT", UpdateCronTasksPath(), token, body)
}

// BuildDeleteCronTaskRequest returns DELETE /api/cron-tasks?id=… with optional Bearer auth.
func BuildDeleteCronTaskRequest(baseURL, token string, id string) (CronRequest, error) {
	if strings.TrimSpace(id) == "" {
		return CronRequest{}, fmt.Errorf("task id is required")
	}
	return build(baseURL, "DELETE", DeleteCronTaskPath(id), token, nil)
}

func build(baseURL, method, path string, token string, body []byte) (CronRequest, error) {
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
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return CronRequest{
		Method:  method,
		URL:     base + path,
		Headers: headers,
		Body:    body,
	}, nil
}

// ConvertLocalCronToUTC converts a simple local 5-field cron to UTC when safe.
// Unsafe patterns (ranges, lists, steps, DST zones with non-fixed offset) error.
// Aligns with CLI convertLocalCronToUTC in cmd/agentcli/cron.go.
func ConvertLocalCronToUTC(expr string, loc *time.Location) (string, error) {
	fields := strings.Fields(strings.TrimSpace(expr))
	if len(fields) != 5 {
		return "", fmt.Errorf("invalid cron expression (want 5 fields)")
	}
	for i, f := range fields {
		if !isSimpleCronToken(f) {
			return "", fmt.Errorf("unsafe cron pattern %q (field %d): cannot auto-convert ranges/lists/steps", f, i+1)
		}
	}
	if !zoneFixedOffset(loc) {
		return "", fmt.Errorf("timezone has DST or variable offset; cannot safely convert cron")
	}

	minStr, hourStr := fields[0], fields[1]
	if minStr == "*" || hourStr == "*" {
		return "", fmt.Errorf("unsafe cron: minute/hour wildcards need manual UTC conversion")
	}
	min, err := strconv.Atoi(minStr)
	if err != nil {
		return "", fmt.Errorf("invalid minute in cron expression")
	}
	hour, err := strconv.Atoi(hourStr)
	if err != nil {
		return "", fmt.Errorf("invalid hour in cron expression")
	}

	now := time.Now().In(loc)
	localT := time.Date(now.Year(), now.Month(), now.Day(), hour, min, 0, 0, loc)
	utcT := localT.UTC()

	if utcT.Day() != localT.Day() || utcT.Month() != localT.Month() || utcT.Year() != localT.Year() {
		if fields[2] != "*" || fields[3] != "*" || fields[4] != "*" {
			return "", fmt.Errorf("unsafe cron: conversion crosses day boundary with constrained date fields")
		}
	}

	return fmt.Sprintf("%d %d %s %s %s", utcT.Minute(), utcT.Hour(), fields[2], fields[3], fields[4]), nil
}

// ConvertUTCCronToLocal converts a simple UTC 5-field cron to local wall time when safe.
// Used on edit open for display; unsafe → error (caller shows stored UTC).
func ConvertUTCCronToLocal(expr string, loc *time.Location) (string, error) {
	fields := strings.Fields(strings.TrimSpace(expr))
	if len(fields) != 5 {
		return "", fmt.Errorf("invalid cron expression (want 5 fields)")
	}
	for i, f := range fields {
		if !isSimpleCronToken(f) {
			return "", fmt.Errorf("unsafe cron pattern %q (field %d): cannot auto-convert ranges/lists/steps", f, i+1)
		}
	}
	if !zoneFixedOffset(loc) {
		return "", fmt.Errorf("timezone has DST or variable offset; cannot safely convert cron")
	}

	minStr, hourStr := fields[0], fields[1]
	if minStr == "*" || hourStr == "*" {
		return "", fmt.Errorf("unsafe cron: minute/hour wildcards need manual local conversion")
	}
	min, err := strconv.Atoi(minStr)
	if err != nil {
		return "", fmt.Errorf("invalid minute in cron expression")
	}
	hour, err := strconv.Atoi(hourStr)
	if err != nil {
		return "", fmt.Errorf("invalid hour in cron expression")
	}

	now := time.Now().UTC()
	utcT := time.Date(now.Year(), now.Month(), now.Day(), hour, min, 0, 0, time.UTC)
	localT := utcT.In(loc)

	if utcT.Day() != localT.Day() || utcT.Month() != localT.Month() || utcT.Year() != localT.Year() {
		if fields[2] != "*" || fields[3] != "*" || fields[4] != "*" {
			return "", fmt.Errorf("unsafe cron: conversion crosses day boundary with constrained date fields")
		}
	}

	return fmt.Sprintf("%d %d %s %s %s", localT.Minute(), localT.Hour(), fields[2], fields[3], fields[4]), nil
}

func isSimpleCronToken(f string) bool {
	if f == "*" {
		return true
	}
	// pure integer only — no '-', ',', '/'
	if strings.ContainsAny(f, "-,/") {
		return false
	}
	_, err := strconv.Atoi(f)
	return err == nil
}

func zoneFixedOffset(loc *time.Location) bool {
	if loc == nil {
		return false
	}
	jan := time.Date(2024, 1, 15, 12, 0, 0, 0, loc)
	jul := time.Date(2024, 7, 15, 12, 0, 0, 0, loc)
	_, off1 := jan.Zone()
	_, off2 := jul.Zone()
	return off1 == off2
}
