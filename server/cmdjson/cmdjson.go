// Package cmdjson runs commands that promise JSON on stdout while keeping
// stderr separate for warnings and diagnostics.
package cmdjson

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Result is the parsed command result. Data comes from stdout JSON; stderr is
// retained separately so warnings do not corrupt JSON parsing.
type Result[T any] struct {
	Data   T
	Stderr string
}

// Warning returns trimmed stderr text.
func (r Result[T]) Warning() string {
	return strings.TrimSpace(r.Stderr)
}

// WarningHeader returns stderr text formatted safely for HTTP headers.
func (r Result[T]) WarningHeader() string {
	return strings.Join(strings.Fields(r.Warning()), " ")
}

// Error reports command execution or JSON decoding failure with captured output.
type Error struct {
	Command string
	Err     error
	Stdout  string
	Stderr  string
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	msg := e.Err.Error()
	if e.Command != "" {
		msg = e.Command + ": " + msg
	}
	if stderr := strings.TrimSpace(e.Stderr); stderr != "" {
		return msg + ": " + stderr
	}
	if stdout := strings.TrimSpace(e.Stdout); stdout != "" {
		return msg + ": " + stdout
	}
	return msg
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// Run executes cmd, decodes JSON from stdout into Result.Data, and returns
// stderr separately in Result.Stderr.
func Run[T any](cmd *exec.Cmd) (Result[T], error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	result := Result[T]{}
	err := cmd.Run()
	result.Stderr = stderr.String()
	if err != nil {
		return result, &Error{Command: commandName(cmd), Err: err, Stdout: stdout.String(), Stderr: result.Stderr}
	}
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &result.Data); err != nil {
		return result, &Error{Command: commandName(cmd), Err: fmt.Errorf("decode stdout JSON: %w", err), Stdout: stdout.String(), Stderr: result.Stderr}
	}
	return result, nil
}

func commandName(cmd *exec.Cmd) string {
	if cmd == nil {
		return ""
	}
	if len(cmd.Args) == 0 {
		return cmd.Path
	}
	return strings.Join(cmd.Args, " ")
}
