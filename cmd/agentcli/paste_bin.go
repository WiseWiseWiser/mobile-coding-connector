package agentcli

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/xhd2015/ai-critic/client"
	"github.com/xhd2015/less-gen/flags"
)

const (
	maxScratchSize  = 256 << 10
	echoThreshold   = 4096
	pasteBinB64Pref = "paste-bin:b64:"
)

const pasteBinHelp = `Usage: remote-agent paste-bin [--json] [--meta] [--read] [-q]

Read or write the File Transfer Quick Transfer scratch pad.

With TTY stdin (default), reads scratch content to stdout. With piped stdin,
reads stdin bytes and writes them to scratch (overwrite). Use --read to force
read mode even when stdin is piped.

Options:
  --json    Output full scratch JSON on stdout (read or write); no preview/echo
  --meta    Print gray timestamp on stderr (updated at / saved at)
  --read    Force read mode; ignore piped stdin for write
  -q        Suppress stdout echo on small writes (≤4096 bytes)

Examples:
  remote-agent paste-bin
  remote-agent paste-bin --json
  echo -n 'hello' | remote-agent paste-bin
  cat file.bin | remote-agent paste-bin --read
`

func runPasteBin(resolve func() (*client.Client, error), args []string) error {
	var jsonOut bool
	var meta bool
	var forceRead bool
	var quiet bool

	args, err := flags.
		Bool("--json", &jsonOut).
		Bool("--meta", &meta).
		Bool("--read", &forceRead).
		Bool("-q,--quiet", &quiet).
		Help("-h,--help", pasteBinHelp).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) > 0 {
		return fmt.Errorf("paste-bin takes no arguments, got %v; see '%s paste-bin --help'", args, active.Name)
	}

	piped, err := isStdinPiped()
	if err != nil {
		return fmt.Errorf("stat stdin: %w", err)
	}

	writeMode := piped && !forceRead

	cli, err := resolve()
	if err != nil {
		return err
	}

	if writeMode {
		return runPasteBinWrite(cli, jsonOut, meta, quiet)
	}
	return runPasteBinRead(cli, jsonOut, meta)
}

func isStdinPiped() (bool, error) {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false, err
	}
	return info.Mode()&os.ModeCharDevice == 0, nil
}

func runPasteBinRead(cli *client.Client, jsonOut, meta bool) error {
	entry, err := cli.GetFileTransferScratch()
	if err != nil {
		return err
	}

	if meta && !jsonOut {
		fmt.Fprintln(os.Stderr, colorGray("updated at "+entry.UpdatedAt))
	}

	if jsonOut {
		data, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("marshal scratch JSON: %w", err)
		}
		if _, err := os.Stdout.Write(data); err != nil {
			return err
		}
		fmt.Println()
		return nil
	}

	content, err := decodeScratchContent(entry.Content)
	if err != nil {
		return fmt.Errorf("decode scratch content: %w", err)
	}
	if len(content) > 0 {
		if _, err := os.Stdout.Write(content); err != nil {
			return err
		}
	}
	return nil
}

func runPasteBinWrite(cli *client.Client, jsonOut, meta, quiet bool) error {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}
	if len(data) > maxScratchSize {
		return fmt.Errorf("content exceeds maximum size of %d bytes", maxScratchSize)
	}

	stored := encodeScratchContent(data)
	entry, err := cli.PutFileTransferScratch(stored)
	if err != nil {
		return err
	}

	if jsonOut {
		out, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("marshal scratch JSON: %w", err)
		}
		if _, err := os.Stdout.Write(out); err != nil {
			return err
		}
		fmt.Println()
		return nil
	}

	n := len(data)
	fmt.Fprintln(os.Stderr, colorGreen(fmt.Sprintf("saved %d bytes", n)))

	if meta {
		fmt.Fprintln(os.Stderr, colorGray("saved at "+entry.UpdatedAt))
	}

	if n > 0 {
		fmt.Fprintln(os.Stderr, colorGray("preview:"))
		writeScratchPreview(os.Stderr, data)
	}

	if !quiet && n > 0 && n <= echoThreshold {
		if _, err := os.Stdout.Write(data); err != nil {
			return err
		}
	}
	return nil
}

func decodeScratchContent(content string) ([]byte, error) {
	if strings.HasPrefix(content, pasteBinB64Pref) {
		return base64.StdEncoding.DecodeString(content[len(pasteBinB64Pref):])
	}
	return []byte(content), nil
}

func encodeScratchContent(data []byte) string {
	if utf8.Valid(data) && bytes.IndexByte(data, 0) < 0 {
		return string(data)
	}
	return pasteBinB64Pref + base64.StdEncoding.EncodeToString(data)
}

func takePreviewRaw(data []byte) (taken []byte, remaining int) {
	const maxPreviewBytes = 200
	const maxPreviewLines = 3

	if len(data) == 0 {
		return nil, 0
	}

	var out []byte
	consumed := 0
	lines := 0

	for consumed < len(data) && lines < maxPreviewLines {
		lineEnd := consumed
		for lineEnd < len(data) && data[lineEnd] != '\n' {
			lineEnd++
		}
		hasNL := lineEnd < len(data)
		lineLen := lineEnd - consumed

		bytesLeft := maxPreviewBytes - len(out)
		if bytesLeft <= 0 {
			break
		}

		take := lineLen
		if take > bytesLeft {
			take = bytesLeft
		}
		out = append(out, data[consumed:consumed+take]...)
		consumed += take

		if take < lineLen {
			break
		}
		if hasNL {
			if len(out) >= maxPreviewBytes {
				break
			}
			out = append(out, '\n')
			consumed++
		}
		lines++
	}

	return out, len(data) - consumed
}

func escapePreviewBytes(data []byte) string {
	var b strings.Builder
	for _, c := range data {
		if c >= 32 && c < 127 {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "\\x%02x", c)
		}
	}
	return b.String()
}

func writeScratchPreview(w io.Writer, data []byte) {
	taken, remaining := takePreviewRaw(data)
	escaped := escapePreviewBytes(taken)
	for _, line := range strings.Split(escaped, "\n") {
		fmt.Fprintf(w, "  %s\n", line)
	}
	if remaining > 0 {
		fmt.Fprintln(w, colorGray(fmt.Sprintf("… (+%d more bytes)", remaining)))
	}
}

func colorGreen(s string) string {
	return "\033[32m" + s + "\033[0m"
}

func colorGray(s string) string {
	return "\033[90m" + s + "\033[0m"
}