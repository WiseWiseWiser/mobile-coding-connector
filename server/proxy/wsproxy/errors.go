package wsproxy

import "fmt"

type ErrorCode string

const (
	ErrNotConfigured   ErrorCode = "NOT_CONFIGURED"
	ErrAlreadyRunning  ErrorCode = "ALREADY_RUNNING"
	ErrNotRunning      ErrorCode = "NOT_RUNNING"
	ErrNoDomain        ErrorCode = "NO_DOMAIN"
	ErrPortInUse       ErrorCode = "PORT_IN_USE"
	ErrXrayUnavailable ErrorCode = "XRAY_UNAVAILABLE"
	ErrStartupFailed   ErrorCode = "STARTUP_FAILED"
	ErrTunnelFailed    ErrorCode = "TUNNEL_FAILED"
	ErrBadRequest      ErrorCode = "BAD_REQUEST"
	ErrInternal        ErrorCode = "INTERNAL"
)

type APIError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func newError(code ErrorCode, msg string) *APIError {
	return &APIError{Code: code, Message: msg}
}
