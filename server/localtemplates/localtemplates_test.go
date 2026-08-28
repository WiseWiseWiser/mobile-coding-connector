package localtemplates

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	libtemplates "github.com/xhd2015/my/lib/templates"
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
	if out.Templates == nil || len(out.Templates) != 0 {
		t.Fatalf("templates=%v", out.Templates)
	}
	if out.MissingRoots == nil || len(out.MissingRoots) != 0 {
		t.Fatalf("missing=%v", out.MissingRoots)
	}
	if out.Roots == nil || len(out.Roots) != 0 {
		t.Fatalf("roots=%v", out.Roots)
	}
}

func TestListRankedAfterUse(t *testing.T) {
	h := handler(t)
	root := filepath.Join(h.Store.ConfigDir, "root")
	writeTemplate(t, filepath.Join(root, "alpha.md"), "alpha body\n")
	writeTemplate(t, filepath.Join(root, "beta.md"), "beta body\n")
	if err := libtemplates.SaveFile(h.Store.ConfigDir, &libtemplates.File{
		TemplateDirs: []libtemplates.TemplateDirEntry{{Path: root, AddedAt: "2026-01-01T00:00:00Z"}},
	}); err != nil {
		t.Fatal(err)
	}

	beta := filepath.Join(root, "beta.md")
	use := serve(h, postUse(beta))
	if use.Code != 200 {
		t.Fatalf("use code=%d body=%s", use.Code, use.Body.String())
	}
	use = serve(h, postUse(beta))
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
	if len(out.Templates) != 2 {
		t.Fatalf("len=%d", len(out.Templates))
	}
	if out.Templates[0].Name != "beta" || out.Templates[0].UseCount != 2 {
		t.Fatalf("first=%+v", out.Templates[0])
	}
	if out.Templates[1].Name != "alpha" || out.Templates[1].UseCount != 0 {
		t.Fatalf("second=%+v", out.Templates[1])
	}
	if out.Templates[0].Body != "beta body" {
		t.Fatalf("body=%q", out.Templates[0].Body)
	}
}

func TestListQueryBody(t *testing.T) {
	h := handler(t)
	root := filepath.Join(h.Store.ConfigDir, "root")
	writeTemplate(t, filepath.Join(root, "sink.md"), `---
name: brainstorm sink
---
/brainstorm following SINK.md about X
`)
	writeTemplate(t, filepath.Join(root, "other.md"), "hello world\n")
	if err := libtemplates.SaveFile(h.Store.ConfigDir, &libtemplates.File{
		TemplateDirs: []libtemplates.TemplateDirEntry{{Path: root, AddedAt: "2026-01-01T00:00:00Z"}},
	}); err != nil {
		t.Fatal(err)
	}
	rec := serve(h, httptest.NewRequest(http.MethodGet, ListPath+"?q=SINK", nil))
	if rec.Code != 200 {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	var out ListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Templates) != 1 || out.Templates[0].Name != "brainstorm sink" {
		t.Fatalf("templates=%+v", out.Templates)
	}
}

func TestUseMissingPath(t *testing.T) {
	h := handler(t)
	rec := serve(h, httptest.NewRequest(http.MethodPost, UsePath, bytes.NewReader([]byte(`{}`))))
	if rec.Code != http.StatusNotFound && rec.Code != http.StatusBadRequest {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAddDirAndCreate(t *testing.T) {
	h := handler(t)
	root := filepath.Join(h.Store.ConfigDir, "prompts")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}

	add := serve(h, postJSON(AddDirPath, AddDirRequest{Path: root, Note: "stubs", NoteSet: true}))
	if add.Code != 200 {
		t.Fatalf("add-dir code=%d body=%s", add.Code, add.Body.String())
	}
	var addOut AddDirResponse
	if err := json.Unmarshal(add.Body.Bytes(), &addOut); err != nil {
		t.Fatal(err)
	}
	if addOut.Duplicate || addOut.Root.Path != root || addOut.Root.Note != "stubs" {
		t.Fatalf("addOut=%+v", addOut)
	}

	dup := serve(h, postJSON(AddDirPath, AddDirRequest{Path: root}))
	if dup.Code != 200 {
		t.Fatalf("dup code=%d body=%s", dup.Code, dup.Body.String())
	}
	if err := json.Unmarshal(dup.Body.Bytes(), &addOut); err != nil {
		t.Fatal(err)
	}
	if !addOut.Duplicate {
		t.Fatal("expected duplicate")
	}

	create := serve(h, postJSON(CreatePath, CreateRequest{
		Name:        "brainstorm sink",
		Description: "Sink X",
		Tags:        []string{"brainstorm", "sink"},
		Body:        "/brainstorm following SINK.md about X",
	}))
	if create.Code != 200 {
		t.Fatalf("create code=%d body=%s", create.Code, create.Body.String())
	}
	var created CreateResponse
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	wantPath := filepath.Join(root, "brainstorm-sink.md")
	if created.Template.Path != wantPath {
		t.Fatalf("path=%q want %q", created.Template.Path, wantPath)
	}
	if created.Template.Name != "brainstorm sink" {
		t.Fatalf("name=%q", created.Template.Name)
	}
	if created.Template.Body != "/brainstorm following SINK.md about X" {
		t.Fatalf("body=%q", created.Template.Body)
	}

	list := serve(h, httptest.NewRequest(http.MethodGet, ListPath, nil))
	var listed ListResponse
	if err := json.Unmarshal(list.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Roots) != 1 || listed.Roots[0].Path != root {
		t.Fatalf("roots=%+v", listed.Roots)
	}
	if len(listed.Templates) != 1 || listed.Templates[0].Path != wantPath {
		t.Fatalf("templates=%+v", listed.Templates)
	}

	conflict := serve(h, postJSON(CreatePath, CreateRequest{
		Name: "brainstorm sink",
		Body: "other",
	}))
	if conflict.Code != http.StatusConflict {
		t.Fatalf("conflict code=%d body=%s", conflict.Code, conflict.Body.String())
	}
}

func TestCreateRequiresRootWhenNone(t *testing.T) {
	h := handler(t)
	rec := serve(h, postJSON(CreatePath, CreateRequest{Name: "x", Body: "y"}))
	if rec.Code != 400 {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAddDirRejectsMissing(t *testing.T) {
	h := handler(t)
	missing := filepath.Join(h.Store.ConfigDir, "no-such-dir")
	rec := serve(h, postJSON(AddDirPath, AddDirRequest{Path: missing}))
	if rec.Code != 400 {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestSlugifyFilename(t *testing.T) {
	if got := slugifyFilename("brainstorm sink"); got != "brainstorm-sink.md" {
		t.Fatalf("got %q", got)
	}
	if got := slugifyFilename("  Hello $AI  "); got != "Hello-AI.md" {
		t.Fatalf("got %q", got)
	}
}

func postJSON(path string, v any) *http.Request {
	body, _ := json.Marshal(v)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestListNeverUsesHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("MY_CONFIG_DIR", filepath.Join(home, "should-not-read"))
	h := handler(t)
	rec := serve(h, httptest.NewRequest(http.MethodGet, ListPath, nil))
	if rec.Code != 200 {
		t.Fatalf("code=%d", rec.Code)
	}
	if _, err := os.Stat(filepath.Join(home, "should-not-read")); !os.IsNotExist(err) {
		t.Fatal("touched MY_CONFIG_DIR")
	}
}

func handler(t *testing.T) *Handler {
	t.Helper()
	fixed := time.Date(2026, 8, 25, 15, 0, 0, 0, time.UTC)
	return &Handler{Store: &libtemplates.Store{
		ConfigDir: t.TempDir(),
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
	req := httptest.NewRequest(http.MethodPost, UsePath, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func writeTemplate(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
