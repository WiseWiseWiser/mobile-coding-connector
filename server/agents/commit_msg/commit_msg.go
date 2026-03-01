package commit_msg

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/xhd2015/lifelog-private/ai-critic/server/gitrunner"
	"github.com/xhd2015/lifelog-private/ai-critic/server/tool_exec"
)

type Logger interface {
	Log(msg string)
	Error(msg string)
}

// Generate produces a commit message for the staged changes in dir.
// It streams progress to the provided Logger and returns the final message.
func Generate(dir string, logger Logger) (string, error) {
	logger.Log("$ git diff --cached")
	stagedDiffOutput, err := gitrunner.DiffCached().Dir(dir).Output()
	if err != nil {
		return "", fmt.Errorf("failed to get staged diff: %w", err)
	}

	stagedDiff := string(stagedDiffOutput)
	if stagedDiff == "" {
		return "", fmt.Errorf("no staged changes to generate commit message for")
	}

	fileCount := strings.Count(stagedDiff, "diff --git")
	if fileCount == 0 && len(stagedDiff) > 0 {
		fileCount = 1
	}

	logger.Log(fmt.Sprintf("Staged files: %d, Diff length: %d chars", fileCount, len(stagedDiff)))
	logger.Log("Passing diff to agent...")

	commitPrompt := fmt.Sprintf(`Generate a brief git commit message (1 line title, max 50 characters, plus a short description if needed) for the following staged changes (git diff). Focus on what changed and why.

Git diff:
%s

Respond with ONLY the commit message in this format:
Title: <short title>
Description: <optional short description>`, stagedDiff)

	logger.Log("$ opencode models")
	freeModels, selectedModel, err := findFreeModel()
	if err != nil {
		logger.Log(fmt.Sprintf("Warning: Could not get free models: %v", err))
	} else {
		logger.Log(fmt.Sprintf("Free models: %s", strings.Join(freeModels, ", ")))
		if selectedModel != "" {
			logger.Log(fmt.Sprintf("Using model: %s", selectedModel))
		}
	}

	promptFile, err := os.CreateTemp("", "commit-prompt-*.txt")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file for prompt: %w", err)
	}
	if _, err := promptFile.WriteString(commitPrompt); err != nil {
		promptFile.Close()
		return "", fmt.Errorf("failed to write prompt to temp file: %w", err)
	}
	promptFile.Close()

	inlineMsg := "Read the attached file and follow the instructions in it."
	// --file passes the prompt via a temp file for two reasons:
	// 1. Avoids exceeding OS argument length limits — large diffs inlined as
	//    CLI arguments cause "argument list too long" failures.
	// 2. Avoids external_directory permission errors — opencode auto-rejects
	//    reads from paths it doesn't recognize when the content is inlined.
	args := []string{"run", inlineMsg, "--file", promptFile.Name()}
	if selectedModel != "" {
		args = append(args, "--model", selectedModel)
	}
	args = append(args, "--format", "json")

	promptSummaryHintLog := fmt.Sprintf("Generate brief git commit message for %d staged file(s), %d chars", fileCount, len(stagedDiff))
	logger.Log(fmt.Sprintf("$ opencode %s  [prompt: %s, file: %s]", shellJoinArgs(args), promptSummaryHintLog, promptFile.Name()))

	logger.Log("Running agent...")

	cmd, err := tool_exec.New("opencode", args, &tool_exec.Options{Dir: dir})
	if err != nil {
		return "", fmt.Errorf("failed to run opencode: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start opencode: %w", err)
	}

	var fullOutput strings.Builder
	doneChan := make(chan struct{})

	go func() {
		pipeToLogger(stderr, logger)
	}()

	go func() {
		buf := make([]byte, 1024)
		for {
			n, readErr := stdout.Read(buf)
			if n > 0 {
				chunk := string(buf[:n])
				fullOutput.WriteString(chunk)
				logger.Log(chunk)
			}
			if readErr != nil {
				break
			}
		}
		doneChan <- struct{}{}
	}()

	<-doneChan
	err = cmd.Wait()
	output := fullOutput.String()

	if err != nil {
		return "", fmt.Errorf("agent failed: %w", err)
	}

	commitMessage := parseOpencodeJSONOutput(output)
	if commitMessage == "" {
		return "", fmt.Errorf("failed to parse commit message from opencode output")
	}

	commitMessage = stripCommitHeaders(commitMessage)

	return commitMessage, nil
}

func stripCommitHeaders(msg string) string {
	lines := strings.Split(msg, "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		for _, prefix := range []string{"Title:", "title:", "Description:", "description:"} {
			if strings.HasPrefix(trimmed, prefix) {
				trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
				break
			}
		}
		result = append(result, trimmed)
	}
	return strings.TrimSpace(strings.Join(result, "\n"))
}

func findFreeModel() (freeModels []string, selectedModel string, err error) {
	cmd, err := tool_exec.New("opencode", []string{"models"}, nil)
	if err != nil {
		return nil, "", err
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, "", err
	}

	models := strings.Split(string(output), "\n")
	for _, model := range models {
		model = strings.TrimSpace(model)
		if strings.Contains(model, "free") || strings.HasPrefix(model, "opencode/") && strings.Contains(model, "-free") {
			freeModels = append(freeModels, model)
		}
	}
	if len(freeModels) > 0 {
		selectedModel = freeModels[0]
	}
	return freeModels, selectedModel, nil
}

func parseOpencodeJSONOutput(output string) string {
	lines := strings.Split(output, "\n")
	var currentStepText strings.Builder
	var lastStopText string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var event map[string]interface{}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		eventType, _ := event["type"].(string)
		part, _ := event["part"].(map[string]interface{})
		if part == nil {
			continue
		}

		switch eventType {
		case "step_start":
			currentStepText.Reset()
		case "text":
			if text, ok := part["text"].(string); ok {
				currentStepText.WriteString(text)
			}
		case "step_finish":
			text := currentStepText.String()
			if text != "" {
				lastStopText = text
			}
		}
	}

	return strings.TrimSpace(lastStopText)
}

func shellJoinArgs(args []string) string {
	quoted := make([]string, len(args))
	for i, a := range args {
		if strings.ContainsAny(a, " \t\n\"'\\") {
			quoted[i] = fmt.Sprintf("%q", a)
		} else {
			quoted[i] = a
		}
	}
	return strings.Join(quoted, " ")
}

func pipeToLogger(r io.Reader, logger Logger) {
	buf := make([]byte, 1024)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			logger.Log(string(buf[:n]))
		}
		if err != nil {
			break
		}
	}
}
