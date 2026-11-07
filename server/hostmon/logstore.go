package hostmon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func appendSample(dir string, sample Sample) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	path := filepath.Join(dir, logName)
	if err := rotateIfNeeded(path); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := json.Marshal(sample)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(data, '\n')); err != nil {
		return err
	}
	return f.Sync()
}

func rotateIfNeeded(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.Size() < MaxLogBytes {
		return nil
	}
	backup := path + ".1"
	_ = os.Remove(backup)
	return os.Rename(path, backup)
}

// readSamples returns the last n samples across the rotated file and the
// current file, oldest first. n <= 0 means the whole history.
func readSamples(dir string, n int) ([]Sample, error) {
	var lines []string
	for _, name := range []string{logName + ".1", logName} {
		path := filepath.Join(dir, name)
		f, err := os.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for sc.Scan() {
			line := sc.Text()
			if line == "" {
				continue
			}
			lines = append(lines, line)
		}
		closeErr := f.Close()
		if err := sc.Err(); err != nil {
			return nil, err
		}
		if closeErr != nil {
			return nil, closeErr
		}
	}
	if n > 0 && len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	out := make([]Sample, 0, len(lines))
	for _, line := range lines {
		var sample Sample
		if err := json.Unmarshal([]byte(line), &sample); err != nil {
			return nil, fmt.Errorf("hostmon log: %w", err)
		}
		out = append(out, sample)
	}
	return out, nil
}

// readLastSample reads only the tail of the current log, so callers polling
// status every couple of seconds do not rescan an 8 MB file.
func readLastSample(dir string) (*Sample, error) {
	f, err := os.Open(filepath.Join(dir, logName))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	const tailBytes = 64 * 1024
	offset := info.Size() - tailBytes
	if offset < 0 {
		offset = 0
	}
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}
	reader := bufio.NewReader(f)
	if offset > 0 {
		// Drop the truncated first line.
		if _, err := reader.ReadString('\n'); err != nil && err != io.EOF {
			return nil, err
		}
	}
	var last string
	for {
		line, readErr := reader.ReadString('\n')
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			last = trimmed
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, readErr
		}
	}
	if last == "" {
		return nil, nil
	}
	var sample Sample
	if err := json.Unmarshal([]byte(last), &sample); err != nil {
		return nil, fmt.Errorf("hostmon log: %w", err)
	}
	return &sample, nil
}

func logBytes(dir string) int64 {
	info, err := os.Stat(filepath.Join(dir, logName))
	if err != nil {
		return 0
	}
	return info.Size()
}
