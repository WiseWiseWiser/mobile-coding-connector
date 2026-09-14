package agentcli

import (
	"bytes"
	"io"
)

// indentingWriter prefixes each complete line written to w with indent.
// Partial Write calls are buffered until a newline.
type indentingWriter struct {
	w      io.Writer
	indent string
	buf    []byte
	atBOL  bool
}

func newIndentingWriter(w io.Writer, indent string) *indentingWriter {
	return &indentingWriter{w: w, indent: indent, atBOL: true}
}

func (iw *indentingWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	n := 0
	for len(p) > 0 {
		i := bytes.IndexByte(p, '\n')
		if i < 0 {
			if err := iw.writeFragment(p); err != nil {
				return n, err
			}
			n += len(p)
			return n, nil
		}
		if err := iw.writeFragment(p[:i+1]); err != nil {
			return n, err
		}
		n += i + 1
		p = p[i+1:]
	}
	return n, nil
}

func (iw *indentingWriter) writeFragment(p []byte) error {
	if len(p) == 0 {
		return nil
	}
	if iw.atBOL {
		if _, err := io.WriteString(iw.w, iw.indent); err != nil {
			return err
		}
	}
	if _, err := iw.w.Write(p); err != nil {
		return err
	}
	iw.atBOL = p[len(p)-1] == '\n'
	return nil
}
