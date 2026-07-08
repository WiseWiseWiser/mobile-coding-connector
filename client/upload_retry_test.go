package client

import (
	"errors"
	"net/http"
	"testing"
)

func TestIsRetryableUploadError_HTTPStatus(t *testing.T) {
	cases := []struct {
		code int
		want bool
	}{
		{http.StatusBadGateway, true},
		{http.StatusServiceUnavailable, true},
		{http.StatusGatewayTimeout, true},
		{http.StatusTooManyRequests, true},
		{http.StatusInternalServerError, true},
		{http.StatusBadRequest, false},
		{http.StatusNotFound, false},
	}
	for _, tc := range cases {
		err := &uploadHTTPError{statusCode: tc.code, status: http.StatusText(tc.code)}
		if got := IsRetryableUploadError(err); got != tc.want {
			t.Errorf("status %d: got %v want %v", tc.code, got, tc.want)
		}
	}
}

func TestIsRetryableUploadError_Nil(t *testing.T) {
	if IsRetryableUploadError(nil) {
		t.Fatal("nil error should not be retryable")
	}
}

func TestIsRetryableUploadError_Unknown(t *testing.T) {
	err := errors.New("connection reset by peer")
	if !IsRetryableUploadError(err) {
		t.Fatal("connection reset should be retryable")
	}
}

func TestResolvedChunkRetry_MaxAttempts(t *testing.T) {
	opts := UploadOptions{ChunkRetry: &ChunkRetryConfig{MaxAttempts: 1}}
	if got := opts.resolvedChunkRetry().maxAttempts; got != 1 {
		t.Fatalf("MaxAttempts=1: got %d want 1", got)
	}
	opts.ChunkRetry.MaxAttempts = 0
	if got := opts.resolvedChunkRetry().maxAttempts; got != defaultChunkMaxAttempts {
		t.Fatalf("MaxAttempts=0: got %d want default %d", got, defaultChunkMaxAttempts)
	}
}