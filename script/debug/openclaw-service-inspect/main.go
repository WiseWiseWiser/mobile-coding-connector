// Command openclaw-service-inspect verifies the remote managed openclaw service
// starts cleanly after deploy. Exit 0 when healthy; non-zero with PASS/FAIL output.
//
// Usage (from repo root):
//
//	go run ./script/debug/openclaw-service-inspect
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xhd2015/ai-critic/client"
)

const serviceName = "openclaw"

func main() {
	if err := run(); err != nil {
		fmt.Printf("FAIL: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("PASS: openclaw service is running with healthy logs")
}

func run() error {
	c, err := loadClient()
	if err != nil {
		return err
	}

	if _, err := c.Ping(); err != nil {
		return fmt.Errorf("remote ping: %w", err)
	}

	services, err := c.ListServices("")
	if err != nil {
		return fmt.Errorf("list services: %w", err)
	}

	var svc *client.ServiceStatus
	for i := range services {
		if services[i].Name == serviceName {
			svc = &services[i]
			break
		}
	}
	if svc == nil {
		return fmt.Errorf("service %q not found on remote", serviceName)
	}

	fmt.Printf("service: %s (%s)\n", svc.Name, svc.ID)
	fmt.Printf("status:  %s\n", svc.Status)
	fmt.Printf("pid:     %d\n", svc.PID)
	fmt.Printf("workdir: %s\n", svc.WorkingDir)
	if svc.LastExitError != "" {
		fmt.Printf("last_exit_error: %s\n", svc.LastExitError)
	}

	if svc.Status != "running" {
		return fmt.Errorf("status is %q, want running", svc.Status)
	}
	if svc.PID <= 0 {
		return fmt.Errorf("pid unavailable while status is running")
	}

	home, err := c.GetHome()
	if err != nil {
		return fmt.Errorf("get home: %w", err)
	}

	logPath := svc.LogPath
	if !filepath.IsAbs(logPath) {
		logPath = filepath.Join(home.Home, logPath)
	}

	var logTail strings.Builder
	code, err := c.Exec(client.ExecRequest{
		Argv: []string{"sh", "-c", fmt.Sprintf("tail -n 40 %s", shellQuote(logPath))},
	}, func(ev client.ExecEvent) {
		if ev.Type == "stdout" || ev.Type == "stderr" {
			logTail.WriteString(ev.Data)
		}
	})
	if err != nil {
		return fmt.Errorf("tail log %s: %w", logPath, err)
	}
	if code != 0 {
		return fmt.Errorf("tail log exited %d", code)
	}

	tail := logTail.String()
	fmt.Printf("log_path: %s\n", logPath)
	fmt.Println("--- log tail ---")
	fmt.Print(tail)
	fmt.Println("--- end log tail ---")

	if strings.Contains(tail, "fork/exec /bin/bash: no such file or directory") {
		return fmt.Errorf("log contains bash fork/exec error (often means WorkingDir is missing)")
	}
	if strings.Contains(tail, "failed to start: fork/exec /bin/bash") {
		return fmt.Errorf("log contains failed bash start marker")
	}

	if strings.Contains(tail, "[gateway] ready") || strings.Contains(tail, "gateway] ready") {
		return nil
	}
	if strings.Contains(tail, "[gateway] starting") || strings.Contains(tail, "gateway] starting HTTP server") {
		return nil
	}

	// Accept running status when the latest tail has no wrapper failure after the last start.
	lastBlock := tail
	if idx := strings.LastIndex(tail, "starting service openclaw"); idx >= 0 {
		lastBlock = tail[idx:]
	}
	if strings.Contains(lastBlock, "failed to start:") {
		return fmt.Errorf("most recent start attempt failed: %s", strings.TrimSpace(lastBlock))
	}

	return fmt.Errorf("log does not show gateway ready/starting in recent tail")
}

func loadClient() (*client.Client, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(home, ".ai-critic", "remote-agent-config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var cfg struct {
		Default string `json:"default"`
		Domains []struct {
			Server string `json:"server"`
			Token  string `json:"token"`
		} `json:"domains"`
		Server string `json:"server"`
		Token  string `json:"token"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	server := strings.TrimRight(strings.TrimSpace(cfg.Server), "/")
	token := cfg.Token
	if server == "" && cfg.Default != "" {
		target := strings.TrimRight(strings.TrimSpace(cfg.Default), "/")
		for _, d := range cfg.Domains {
			if strings.TrimRight(strings.TrimSpace(d.Server), "/") == target {
				server = target
				token = d.Token
				break
			}
		}
	}
	if server == "" && len(cfg.Domains) > 0 {
		server = strings.TrimRight(strings.TrimSpace(cfg.Domains[0].Server), "/")
		token = cfg.Domains[0].Token
	}
	if server == "" {
		return nil, fmt.Errorf("no server in %s", path)
	}
	return client.New(server, token), nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}