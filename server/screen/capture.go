package screen

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// source yields JPEG frames, one per call to next.
type source interface {
	next(ctx context.Context) ([]byte, error)
	close()
}

// errCaptureStopped marks a capture error caused by shutting down rather than a
// real failure, so the loop can exit quietly.
var errCaptureStopped = errors.New("screen: capture stopped")

// captureStartTimeout bounds how long the capture command may stay silent
// before we call it broken. ffmpeg hangs indefinitely when another app owns
// the screen capture device, so without a deadline the UI would sit on
// "starting" forever.
var captureStartTimeout = 10 * time.Second

// startSourceFunc is a seam for tests that need to fake a stalled capture.
var startSourceFunc = startSource

// reapStale is swapped out by tests so a unit run never touches real processes.
var reapStale = reapStaleCapture

// startSource launches the platform capture command for cfg.
func startSource(ctx context.Context, maxWidth, fps, quality int) (source, error) {
	return startPlatformSource(ctx, maxWidth, fps, quality)
}

// processSource streams MJPEG from a child process: ffmpeg on every platform,
// plus the ScreenCaptureKit helper on macOS.
type processSource struct {
	label  string
	cmd    *exec.Cmd
	out    io.ReadCloser
	br     *bufio.Reader
	stderr *tailBuffer

	closeOnce sync.Once
}

// startProcess spawns bin and returns a source that reads its MJPEG stdout.
func startProcess(ctx context.Context, label, bin string, args []string) (*processSource, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.SysProcAttr = sysProcAttr()
	stderr := &tailBuffer{max: 4096}
	cmd.Stderr = stderr
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", label, err)
	}
	return &processSource{
		label:  label,
		cmd:    cmd,
		out:    out,
		br:     bufio.NewReaderSize(out, 256*1024),
		stderr: stderr,
	}, nil
}

func startFFmpeg(ctx context.Context, args []string) (*processSource, error) {
	bin, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, fmt.Errorf("ffmpeg not found in PATH: install it with `brew install ffmpeg` (macOS) or `apt install ffmpeg` (Debian/Ubuntu)")
	}
	return startProcess(ctx, "ffmpeg", bin, args)
}

func (s *processSource) next(ctx context.Context) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, errCaptureStopped
	}
	frame, err := readJPEG(s.br)
	if err != nil {
		if ctx.Err() != nil {
			return nil, errCaptureStopped
		}
		if msg := s.errorLine(); msg != "" {
			return nil, fmt.Errorf("capture failed: %s", msg)
		}
		return nil, fmt.Errorf("capture failed: %w", err)
	}
	return frame, nil
}

func (s *processSource) close() {
	s.closeOnce.Do(func() {
		if s.cmd.Process != nil {
			_ = s.cmd.Process.Kill()
		}
		_ = s.out.Close()
		done := make(chan struct{})
		go func() {
			_ = s.cmd.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
	})
}

// errorLine returns the last diagnostic from the child's stderr, ignoring its
// informational banner: a capture that is killed reports nothing useful, and
// "capture failed: ai-critic-screencap: info: capturing ..." would hide that.
func (s *processSource) errorLine() string {
	lines := strings.Split(strings.TrimSpace(s.stderr.String()), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.Contains(line, ": info: ") {
			continue
		}
		return line
	}
	return ""
}

// captureFault is the guidance appended to capture errors that happen before
// the first frame, where the cause is almost always environmental.
const captureFault = "check that the display is awake, that no other app is capturing it, and that Screen Recording permission is still granted"

// captureStartError explains a capture command that never produced a frame.
func captureStartError(src source) error {
	return fmt.Errorf("screen capture produced no frames in %s (%s)", captureStartTimeout, captureFault)
}

// captureFirstFrameError explains a capture command that exited or failed
// before delivering any frame.
func captureFirstFrameError(src source, err error) error {
	if fs, ok := src.(*processSource); ok {
		if tail := fs.errorLine(); tail != "" {
			return fmt.Errorf("screen capture failed before the first frame: %s (%s)", tail, captureFault)
		}
	}
	return fmt.Errorf("screen capture failed before the first frame: %v (%s)", err, captureFault)
}

// readJPEG scans the MJPEG stream for one complete JPEG image. 0xFFD9 cannot
// appear inside entropy-coded data (it would be byte-stuffed), so the scan is
// safe and avoids parsing JPEG segments.
func readJPEG(br *bufio.Reader) ([]byte, error) {
	var out bytes.Buffer
	started := false
	var prev byte
	for {
		b, err := br.ReadByte()
		if err != nil {
			if err == io.EOF && out.Len() == 0 {
				return nil, io.EOF
			}
			return nil, err
		}
		if !started {
			if prev == 0xFF && b == 0xD8 {
				started = true
				out.Grow(64 * 1024)
				out.Write([]byte{0xFF, 0xD8})
			}
			prev = b
			continue
		}
		out.WriteByte(b)
		if prev == 0xFF && b == 0xD9 {
			return out.Bytes(), nil
		}
		prev = b
	}
}

// tailBuffer keeps the last max bytes written, so a failing ffmpeg invocation
// can be reported without unbounded memory growth.
type tailBuffer struct {
	mu  sync.Mutex
	max int
	buf []byte
}

func (t *tailBuffer) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buf = append(t.buf, p...)
	if len(t.buf) > t.max {
		t.buf = t.buf[len(t.buf)-t.max:]
	}
	return len(p), nil
}

func (t *tailBuffer) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return string(t.buf)
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}
