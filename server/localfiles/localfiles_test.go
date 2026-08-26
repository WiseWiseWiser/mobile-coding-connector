package localfiles

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/xhd2015/my/libfiles"
)

func TestListEmpty(t *testing.T) {
	h := handler(t)
	rec := serve(h, httptest.NewRequest(http.MethodGet, ListPath, nil))
	if rec.Code != 200 {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	var out ListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Files == nil || len(out.Files) != 0 {
		t.Fatalf("files=%v", out.Files)
	}
}

func TestListRankedAfterUse(t *testing.T) {
	h := handler(t)
	a := filepath.Join(h.Store.ConfigDir, "a.md")
	b := filepath.Join(h.Store.ConfigDir, "b.md")
	if _, _, err := h.Store.Add(a, "", false); err != nil {
		t.Fatal(err)
	}
	if _, _, err := h.Store.Add(b, "", false); err != nil {
		t.Fatal(err)
	}

	use := serve(h, postUse(b))
	if use.Code != 200 {
		t.Fatalf("use code=%d body=%s", use.Code, use.Body.String())
	}
	use = serve(h, postUse(b))
	if use.Code != 200 {
		t.Fatalf("use2 code=%d body=%s", use.Code, use.Body.String())
	}

	rec := serve(h, httptest.NewRequest(http.MethodGet, ListPath, nil))
	if rec.Code != 200 {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	var out ListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Files) != 2 {
		t.Fatalf("len=%d", len(out.Files))
	}
	if out.Files[0].Path != b || out.Files[0].UseCount != 2 {
		t.Fatalf("first=%+v", out.Files[0])
	}
	if out.Files[1].Path != a || out.Files[1].UseCount != 0 {
		t.Fatalf("second=%+v", out.Files[1])
	}
}

func TestListQueryPathAndMissing(t *testing.T) {
	h := handler(t)
	missing := filepath.Join(h.Store.ConfigDir, "gone", "draft.md")
	other := filepath.Join(h.Store.ConfigDir, "other.txt")
	if _, _, err := h.Store.Add(missing, "future draft", true); err != nil {
		t.Fatal(err)
	}
	if _, _, err := h.Store.Add(other, "", false); err != nil {
		t.Fatal(err)
	}
	rec := serve(h, httptest.NewRequest(http.MethodGet, ListPath+"?q=draft", nil))
	if rec.Code != 200 {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	var out ListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Files) != 1 || out.Files[0].Path != missing {
		t.Fatalf("files=%+v", out.Files)
	}
	if out.Files[0].Exists {
		t.Fatal("expected missing path Exists=false")
	}
	if out.Files[0].Note != "future draft" {
		t.Fatalf("note=%q", out.Files[0].Note)
	}
}

func TestUseUnknown(t *testing.T) {
	h := handler(t)
	rec := serve(h, postUse(filepath.Join(h.Store.ConfigDir, "nope.md")))
	if rec.Code != 404 {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
}

func handler(t *testing.T) *Handler {
	t.Helper()
	cfg := t.TempDir()
	fixed := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	return &Handler{Store: &libfiles.Store{
		ConfigDir: cfg,
		Now:       func() time.Time { return fixed },
	}}
}

func serve(h *Handler, req *http.Request) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	Register(mux, h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func postUse(path string) *http.Request {
	body, _ := json.Marshal(UseRequest{Path: path})
	return httptest.NewRequest(http.MethodPost, UsePath, bytes.NewReader(body))
}
