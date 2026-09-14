package agentcli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xhd2015/ai-critic/client"
)

func TestUploadDirStagePrinter_LiveSpine(t *testing.T) {
	var errBuf bytes.Buffer
	var outBuf bytes.Buffer
	p := &uploadDirStagePrinter{errW: &errBuf}
	// Capture stdout product via temporary swap is awkward; call PrintProduct
	// after redirecting fmt to outBuf by testing stderr spine only here.

	p.OnProgress(client.UploadDirProgress{
		Phase:        client.UploadDirPhaseResolved,
		RelativePath: "/tmp/apps/srcdir",
		TotalItems:   3,
		TotalBytes:   1200,
	})
	p.OnProgress(client.UploadDirProgress{Phase: client.UploadDirPhasePacking})
	p.OnProgress(client.UploadDirProgress{Phase: client.UploadDirPhasePacking, FileSize: 450})
	p.OnProgress(client.UploadDirProgress{Phase: client.UploadDirPhaseUploading, FileSize: 450})
	p.OnProgress(client.UploadDirProgress{
		Phase: client.UploadDirPhaseUploading,
		Chunk: client.UploadProgress{
			ChunkIndex:     0,
			TotalChunks:    1,
			CompletedBytes: 450,
			TotalBytes:     450,
			Phase:          client.UploadChunkUploaded,
		},
	})
	p.OnProgress(client.UploadDirProgress{
		Phase:        client.UploadDirPhaseApplying,
		RelativePath: "/tmp/apps/srcdir",
	})
	p.FinishApply([]string{"a.txt"})

	stderr := errBuf.String()
	for _, want := range []string{
		"[1/4] resolve      /tmp/apps/srcdir",
		"notice: 3 items, 1.20 KB",
		"[2/4] pack         tar.xz",
		"notice: archive 450 B",
		"[3/4] upload       450 B",
		"chunk 1/1 uploaded",
		"[4/4] apply        /tmp/apps/srcdir",
		"warning: overridden 1 existing file(s)",
		"warning:   a.txt",
		"ok",
	} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr missing %q;\n%s", want, stderr)
		}
	}
	// Exactly four stage markers.
	if c := strings.Count(stderr, "[1/4]"); c != 1 {
		t.Fatalf("[1/4] count=%d", c)
	}
	if c := strings.Count(stderr, "[2/4]"); c != 1 {
		t.Fatalf("[2/4] count=%d", c)
	}
	if c := strings.Count(stderr, "[3/4]"); c != 1 {
		t.Fatalf("[3/4] count=%d", c)
	}
	if c := strings.Count(stderr, "[4/4]"); c != 1 {
		t.Fatalf("[4/4] count=%d", c)
	}
	_ = outBuf
}

func TestUploadDirStagePrinter_DryRunWould(t *testing.T) {
	var errBuf bytes.Buffer
	p := &uploadDirStagePrinter{dryRun: true, errW: &errBuf}
	p.OnProgress(client.UploadDirProgress{
		Phase:        client.UploadDirPhaseResolved,
		RelativePath: "/tmp/apps/srcdir",
		TotalBytes:   100,
	})
	p.OnProgress(client.UploadDirProgress{Phase: client.UploadDirPhasePacking})
	p.OnProgress(client.UploadDirProgress{Phase: client.UploadDirPhaseUploading})
	p.OnProgress(client.UploadDirProgress{
		Phase:        client.UploadDirPhaseApplying,
		RelativePath: "/tmp/apps/srcdir",
	})
	p.FinishApply(nil)

	stderr := errBuf.String()
	for _, want := range []string{
		"[1/4] resolve",
		"[2/4] pack",
		"would: pack tar.xz",
		"[3/4] upload",
		"would: upload archive",
		"[4/4] apply",
		"would: apply extract/merge",
	} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr missing %q;\n%s", want, stderr)
		}
	}
	if strings.Contains(stderr, "\nok\n") || strings.HasSuffix(stderr, "\nok") {
		t.Fatalf("dry-run should not print ok;\n%s", stderr)
	}
}

func TestIndentingWriter_PrefixesLines(t *testing.T) {
	var buf bytes.Buffer
	w := newIndentingWriter(&buf, "      ")
	_, _ = w.Write([]byte("hello\nworld"))
	_, _ = w.Write([]byte("!\n"))
	got := buf.String()
	want := "      hello\n      world!\n"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
