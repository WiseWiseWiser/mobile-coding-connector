# Client Download Resilience Doctests

Unit tests for `client.Client.DownloadFile` retry on transient failures and HTTP
Range resume for partial local files.

# DSN (Domain Specific Notion)

The download resilience harness models single-file transfer from CLI client to
ai-critic server file-download API.

**Participants**

- **Client.DownloadFile** — GET `/api/files/download`; writes local file with optional Range resume.
- **Download retry policy** — classifies errors as retryable; re-issues GET with backoff.
- **Mock download server** — `httptest.Server` implementing `/api/files/download` with Range support.
- **Attempt tracker** — counts GET attempts and recorded `Range` headers.

**Behaviors**

- Transient 502/503/504 or transport errors trigger retry on the same file transfer.
- Non-retryable 4xx abort immediately without further attempts.
- After retries exhausted, download fails with last error.
- Partial local file triggers `Range: bytes=<offset>-` on resume; bytes append to existing file.

## Version

0.0.2

## Decision Tree

```
[DownloadFile retry + resume]
 |
 +-- transient-recovery/
 |    |
 |    +-- succeeds-after-502/                  (LEAF)  flaky GET recovers; full file OK
 |
 +-- retry-exhaustion/
 |    |
 |    +-- aborts-after-max-attempts/   (LEAF)  always-502 until cap
 |
 +-- resume/
      |
      +-- partial-file-range-request/  (LEAF)  Range header + append bytes
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `transient-recovery/succeeds-after-502` | GET fails twice with 502, succeeds on 3rd try; file intact |
| 2 | `retry-exhaustion/aborts-after-max-attempts` | Permanent 502 exhausts attempts and errors |
| 3 | `resume/partial-file-range-request` | Pre-filled local file sends Range; assembled bytes match |

## How to Run

```sh
doctest vet ./client/tests/download-resilience
doctest test ./client/tests/download-resilience/...
```

```go
import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/xhd2015/ai-critic/client"
)

type Request struct {
	FileSize            int64
	LocalPrefillBytes   int64
	TransientFails      int
	FailStatus          int
	MaxDownloadAttempts int
	AlwaysFail          bool
	RemotePath          string
}

type Response struct {
	DownloadErr        string
	ResultPath         string
	ResultSize         int64
	DownloadAttempts   int
	RangeHeaders       []string
	LocalFileContent   []byte
	WantContent        []byte
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}

	if req.FileSize <= 0 {
		req.FileSize = 4096
	}
	if req.FailStatus == 0 {
		req.FailStatus = http.StatusBadGateway
	}
	if req.MaxDownloadAttempts == 0 {
		req.MaxDownloadAttempts = 5
	}
	if req.RemotePath == "" {
		req.RemotePath = "/home/remote/data.bin"
	}

	wantContent := make([]byte, req.FileSize)
	for i := range wantContent {
		wantContent[i] = byte((i*7 + 13) % 251)
	}
	resp.WantContent = wantContent

	localFile := filepath.Join(t.TempDir(), "download-dest.bin")
	if req.LocalPrefillBytes > 0 {
		if req.LocalPrefillBytes > req.FileSize {
			return nil, fmt.Errorf("LocalPrefillBytes %d exceeds FileSize %d", req.LocalPrefillBytes, req.FileSize)
		}
		if err := os.WriteFile(localFile, wantContent[:req.LocalPrefillBytes], 0644); err != nil {
			return nil, err
		}
	}

	var mu sync.Mutex
	attempts := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/files/home":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"home":"/home/remote","cwd":"/home/remote"}`))

		case "/api/files/download":
			if r.URL.Query().Get("path") != req.RemotePath {
				http.NotFound(w, r)
				return
			}

			mu.Lock()
			attempts++
			resp.DownloadAttempts = attempts
			if rng := r.Header.Get("Range"); rng != "" {
				resp.RangeHeaders = append(resp.RangeHeaders, rng)
			}
			mu.Unlock()

			if shouldFailDownload(req, attempts) {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(req.FailStatus)
				fmt.Fprintf(w, "error code: %d", req.FailStatus)
				return
			}

			body := wantContent
			status := http.StatusOK
			rangeHdr := r.Header.Get("Range")
			if rangeHdr != "" {
				start, err := parseRangeStart(rangeHdr, req.FileSize)
				if err != nil {
					w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
					return
				}
				body = wantContent[start:]
				status = http.StatusPartialContent
				w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, req.FileSize-1, req.FileSize))
			}
			w.Header().Set("Content-Length", strconv.FormatInt(int64(len(body)), 10))
			w.WriteHeader(status)
			w.Write(body)

		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := client.New(srv.URL, "")
	downloadOpts := client.DownloadOptions{
		Retry: &client.DownloadRetryConfig{
			MaxAttempts: req.MaxDownloadAttempts,
			Backoff:     func(int) int64 { return 0 },
		},
	}
	result, err := c.DownloadFile(req.RemotePath, localFile, downloadOpts, nil)
	if err != nil {
		resp.DownloadErr = err.Error()
	} else if result != nil {
		resp.ResultPath = result.LocalPath
		resp.ResultSize = result.Size
	}

	data, readErr := os.ReadFile(localFile)
	if readErr != nil && !os.IsNotExist(readErr) {
		return nil, readErr
	}
	resp.LocalFileContent = data
	resp.ResultPath = localFile

	t.Logf("evidence: DownloadAttempts=%d RangeHeaders=%v DownloadErr=%q ResultSize=%d LocalLen=%d",
		resp.DownloadAttempts, resp.RangeHeaders, resp.DownloadErr, resp.ResultSize, len(resp.LocalFileContent))
	return resp, nil
}

func shouldFailDownload(req *Request, attempt int) bool {
	if req.AlwaysFail {
		return true
	}
	if req.TransientFails > 0 {
		return attempt <= req.TransientFails
	}
	return false
}

func parseRangeStart(rangeHdr string, total int64) (int64, error) {
	if !strings.HasPrefix(rangeHdr, "bytes=") {
		return 0, fmt.Errorf("unsupported range %q", rangeHdr)
	}
	rest := strings.TrimPrefix(rangeHdr, "bytes=")
	parts := strings.SplitN(rest, "-", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("malformed range %q", rangeHdr)
	}
	start, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}
	if start < 0 || start >= total {
		return 0, fmt.Errorf("range start %d out of bounds for size %d", start, total)
	}
	return start, nil
}
```