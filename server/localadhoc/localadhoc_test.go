package localadhoc

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func serve(h *Handler, req *http.Request) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	Register(mux, h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestGetMissingReturnsEmpty(t *testing.T) {
	h := &Handler{DataDir: t.TempDir()}
	rec := serve(h, httptest.NewRequest(http.MethodGet, Path, nil))
	if rec.Code != 200 {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	var got Response
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Content != "" {
		t.Fatalf("content %q", got.Content)
	}
	if !strings.HasSuffix(got.Path, fileName) {
		t.Fatalf("path %q", got.Path)
	}
}

func TestPutThenGet(t *testing.T) {
	dir := t.TempDir()
	h := &Handler{DataDir: dir}
	rec := serve(h, httptest.NewRequest(http.MethodPut, Path, strings.NewReader(`{"content":"note one"}`)))
	if rec.Code != 200 {
		t.Fatalf("put status %d body %s", rec.Code, rec.Body.String())
	}
	disk, err := os.ReadFile(filepath.Join(dir, fileName))
	if err != nil {
		t.Fatal(err)
	}
	if string(disk) != "note one" {
		t.Fatalf("disk %q", disk)
	}
	rec = serve(h, httptest.NewRequest(http.MethodGet, Path, nil))
	var got Response
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Content != "note one" {
		t.Fatalf("got %+v", got)
	}
}
