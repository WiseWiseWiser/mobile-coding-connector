package localclipboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhd2015/dot-pkgs/go-pkgs/getclipboard"
)

type fakeSource struct {
	image []byte
	text  []byte
}

func (f *fakeSource) Init() error            { return nil }
func (f *fakeSource) ReadImage() []byte      { return f.image }
func (f *fakeSource) ReadText() []byte       { return f.text }
func (f *fakeSource) ExtraFormats() []getclipboard.Format {
	return nil
}

func serve(h *Handler, req *http.Request) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	Register(mux, h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestPeekText(t *testing.T) {
	h := &Handler{Source: &fakeSource{text: []byte("hello world")}}
	rec := serve(h, httptest.NewRequest(http.MethodGet, PeekPath, nil))
	if rec.Code != 200 {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	var got PeekResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Kind != "text" || got.Bytes != 11 || !strings.Contains(got.Preview, "hello") {
		t.Fatalf("got %+v", got)
	}
}

func TestDumpTextWritesFile(t *testing.T) {
	dir := t.TempDir()
	h := &Handler{Source: &fakeSource{text: []byte("dump me")}, DumpDir: dir}
	rec := serve(h, httptest.NewRequest(http.MethodPost, DumpPath, strings.NewReader("{}")))
	if rec.Code != 200 {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	var got DumpResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Kind != "text" || got.Ext != "txt" {
		t.Fatalf("got %+v", got)
	}
	data, err := os.ReadFile(got.Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "dump me" {
		t.Fatalf("file %q", data)
	}
	if filepath.Dir(got.Path) != dir {
		t.Fatalf("path dir %q want %q", filepath.Dir(got.Path), dir)
	}
}

func TestDumpEmptyConflict(t *testing.T) {
	h := &Handler{Source: &fakeSource{}, DumpDir: t.TempDir()}
	rec := serve(h, httptest.NewRequest(http.MethodPost, DumpPath, strings.NewReader("{}")))
	if rec.Code != http.StatusConflict {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
}

func TestPeekEmpty(t *testing.T) {
	h := &Handler{Source: &fakeSource{}}
	rec := serve(h, httptest.NewRequest(http.MethodGet, PeekPath, nil))
	if rec.Code != 200 {
		t.Fatalf("status %d", rec.Code)
	}
	var got PeekResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Kind != "empty" {
		t.Fatalf("got %+v", got)
	}
}
