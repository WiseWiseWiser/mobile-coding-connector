// +build ignore

// Quick test for OpenCode settings using HTTP API and browser automation via chromedp
// This is a skill that can be run with: go run ./skills/quick-test-server-and-frontend
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/chromedp/chromedp"
)

const (
	defaultBaseURL = "https://agent-fast-apex-nest-23aed.xhd2015.xyz"
	settingsPath   = "/project/mobile-coding-connector/agent/opencode/settings"
	apiTimeout     = 30 * time.Second
)

// TestResult represents a single test result
type TestResult struct {
	Name    string      `json:"name"`
	Status  string      `json:"status"` // passed, failed, skipped
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	DurationMs int64    `json:"duration_ms"`
}

// TestReport represents the complete test report
type TestReport struct {
	Timestamp   time.Time    `json:"timestamp"`
	BaseURL     string       `json:"base_url"`
	TotalTests  int          `json:"total_tests"`
	Passed      int          `json:"passed"`
	Failed      int          `json:"failed"`
	Skipped     int          `json:"skipped"`
	Results     []TestResult `json:"results"`
	Screenshots []string     `json:"screenshots,omitempty"`
}

func main() {
	baseURL := os.Getenv("TEST_URL")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	report := &TestReport{
		Timestamp: time.Now(),
		BaseURL:   baseURL,
		Results:   []TestResult{},
	}

	fmt.Printf("🚀 OpenCode Settings Quick Test\n")
	fmt.Printf("   URL: %s%s\n\n", baseURL, settingsPath)

	// Run tests
	runTest(report, "API Health Check", func() (interface{}, error) {
		return testAPIHealth(ctx, baseURL)
	})

	runTest(report, "Settings Page Load", func() (interface{}, error) {
		return testSettingsPage(ctx, baseURL, report)
	})

	runTest(report, "Port Field Detection", func() (interface{}, error) {
		return testPortField(ctx, baseURL, report)
	})

	// Print report
	printReport(report)

	// Save report to file
	reportFile := "/tmp/opencode-test-report.json"
	if err := saveReport(report, reportFile); err != nil {
		log.Printf("Warning: failed to save report: %v", err)
	} else {
		fmt.Printf("\n📄 Report saved: %s\n", reportFile)
	}

	// Exit with appropriate code
	if report.Failed > 0 {
		os.Exit(1)
	}
}

func runTest(report *TestReport, name string, testFunc func() (interface{}, error)) {
	fmt.Printf("📍 %s\n", name)
	start := time.Now()
	
	data, err := testFunc()
	duration := time.Since(start)
	
	result := TestResult{
		Name:       name,
		DurationMs: duration.Milliseconds(),
		Data:       data,
	}
	
	if err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		fmt.Printf("   ❌ Failed: %v\n", err)
	} else {
		result.Status = "passed"
		fmt.Printf("   ✓ Passed (%dms)\n", duration.Milliseconds())
	}
	
	report.Results = append(report.Results, result)
}

func testAPIHealth(ctx context.Context, baseURL string) (interface{}, error) {
	client := &http.Client{Timeout: apiTimeout}
	resp, err := client.Get(baseURL + "/api/health")
	if err != nil {
		return nil, fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	
	return map[string]interface{}{
		"status": resp.Status,
		"code":   resp.StatusCode,
	}, nil
}

func testSettingsPage(ctx context.Context, baseURL string, report *TestReport) (interface{}, error) {
	// Create a browser instance
	browser, err := chromedp.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create browser: %w", err)
	}
	defer browser.Close()

	var pageTitle string
	err = browser.Run(ctx, chromedp.Tasks{
		chromedp.Navigate(baseURL + SETTINGS_PATH),
		chromedp.WaitReady("body"),
		chromedp.Title(&pageTitle),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load page: %w", err)
	}

	// Take screenshot
	timestamp := time.Now().Unix()
	screenshotFile := fmt.Sprintf("%s/page-load-%d.png", SCREENSHOT_DIR, timestamp)
	err = browser.Run(ctx, chromedp.Tasks{
		chromedp.FullScreenshot(&screenshotFile, 90),
	})
	if err != nil {
		fmt.Printf("   Warning: failed to take screenshot: %v\n", err)
	} else {
		report.Screenshots = append(report.Screenshots, screenshotFile)
		fmt.Printf("   📸 Screenshot: %s\n", screenshotFile)
	}

	return map[string]interface{}{
		"title":    pageTitle,
		"url":      baseURL + SETTINGS_PATH,
		"selector": "body loaded",
	}, nil
}

func testPortField(ctx context.Context, baseURL string, report *TestReport) (interface{}, error) {
	browser, err := chromedp.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create browser: %w", err)
	}
	defer browser.Close()

	var foundInputs []map[string]string
	err = browser.Run(ctx, chromedp.Tasks{
		chromedp.Navigate(baseURL + SETTINGS_PATH),
		chromedp.WaitReady("body"),
		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('input')).map(input => ({
				type: input.type,
				name: input.name,
				placeholder: input.placeholder,
				value: input.value,
				id: input.id
			}))
		`, &foundInputs),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find inputs: %w", err)
	}

	// Look for port input
	var portInput map[string]string
	for _, input := range foundInputs {
		if input["type"] == "number" || 
		   (input["name"] && input["name"].includes("port")) ||
		   (input["placeholder"] && input["placeholder"].includes("port")) {
			portInput = input
			break
		}
	}

	// Take screenshot
	timestamp := time.Now().Unix()
	screenshotFile := fmt.Sprintf("%s/port-field-%d.png", SCREENSHOT_DIR, timestamp)
	err = browser.Run(ctx, chromedp.Tasks{
		chromedp.FullScreenshot(&screenshotFile, 90),
	})
	if err == nil {
		report.Screenshots = append(report.Screenshots, screenshotFile)
	}

	return map[string]interface{}{
		"totalInputs": len(foundInputs),
		"portInput":   portInput,
		"allInputs":   foundInputs[:min(len(foundInputs), 10)],
	}, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func printReport(report *TestReport) {
	fmt.Println("")
	fmt.Println("========================================")
	fmt.Println("TEST REPORT")
	fmt.Println("========================================")
	fmt.Printf("Timestamp: %s\n", report.Timestamp.Format(time.RFC3339))
	fmt.Printf("Base URL: %s\n", report.BaseURL)
	fmt.Println("")
	
	report.TotalTests = len(report.Results)
	for _, result := range report.Results {
		switch result.Status {
		case "passed":
			report.Passed++
		case "failed":
			report.Failed++
		default:
			report.Skipped++
		}
	}
	
	fmt.Printf("Total Tests: %d\n", report.TotalTests)
	fmt.Printf("Passed: %d ✓\n", report.Passed)
	fmt.Printf("Failed: %d ✗\n", report.Failed)
	fmt.Printf("Skipped: %d\n", report.Skipped)
	fmt.Println("")
	
	fmt.Println("Test Details:")
	for i, result := range report.Results {
		statusIcon := "✓"
		if result.Status == "failed" {
			statusIcon = "✗"
		}
		fmt.Printf("  %s %d. %s (%dms)\n", statusIcon, i+1, result.Name, result.DurationMs)
		if result.Error != "" {
			fmt.Printf("     Error: %s\n", result.Error)
		}
	}
	
	if len(report.Screenshots) > 0 {
		fmt.Println("")
		fmt.Println("Screenshots:")
		for _, screenshot := range report.Screenshots {
			fmt.Printf("  - %s\n", screenshot)
		}
	}
	
	fmt.Println("")
	fmt.Println("========================================")
	
	if report.Failed > 0 {
		fmt.Println("RESULT: FAILED")
	} else {
		fmt.Println("RESULT: PASSED")
	}
	fmt.Println("========================================")
}

func saveReport(report *TestReport, filename string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}
