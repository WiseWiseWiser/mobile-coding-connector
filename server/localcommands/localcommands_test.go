package localcommands

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	libcommands "github.com/xhd2015/my/lib/commands"
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
	if out.Commands == nil || len(out.Commands) != 0 {
		t.Fatalf("commands=%v", out.Commands)
	}
}

func TestListRankedAfterUse(t *testing.T) {
	h := handler(t)
	a := "echo a"
	b := "echo b"
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
	if len(out.Commands) != 2 {
		t.Fatalf("len=%d", len(out.Commands))
	}
	if out.Commands[0].Command != b || out.Commands[0].UseCount != 2 {
		t.Fatalf("first=%+v", out.Commands[0])
	}
	if out.Commands[1].Command != a || out.Commands[1].UseCount != 0 {
		t.Fatalf("second=%+v", out.Commands[1])
	}
}

func TestListQueryNoteAndCommand(t *testing.T) {
	h := handler(t)
	save := "kool iterm2 sessions save --file ~/tmp/iterm2-session-spaces-all.json"
	other := "echo other"
	if _, _, err := h.Store.Add(save, "iTerm2 save sessions", true); err != nil {
		t.Fatal(err)
	}
	if _, _, err := h.Store.Add(other, "", false); err != nil {
		t.Fatal(err)
	}
	rec := serve(h, httptest.NewRequest(http.MethodGet, ListPath+"?q=save", nil))
	if rec.Code != 200 {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	var out ListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Commands) != 1 || out.Commands[0].Command != save {
		t.Fatalf("commands=%+v", out.Commands)
	}
	if out.Commands[0].Note != "iTerm2 save sessions" {
		t.Fatalf("note=%q", out.Commands[0].Note)
	}
}

func TestUseUnknown(t *testing.T) {
	h := handler(t)
	rec := serve(h, postUse("echo nope"))
	if rec.Code != 404 {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAddNewAndDuplicateNote(t *testing.T) {
	h := handler(t)
	cmd := "kool iterm2 tab-set run services"
	rec := serve(h, postAdd(AddRequest{Command: cmd, Note: "tab-set", NoteSet: true}))
	if rec.Code != 200 {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	var out AddResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Duplicate || out.Command.Command != cmd || out.Command.Note != "tab-set" {
		t.Fatalf("out=%+v", out)
	}

	rec = serve(h, postAdd(AddRequest{Command: cmd, Note: "renamed", NoteSet: true}))
	if rec.Code != 200 {
		t.Fatalf("dup code=%d body=%s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if !out.Duplicate || out.Command.Note != "renamed" {
		t.Fatalf("dup out=%+v", out)
	}

	list := serve(h, httptest.NewRequest(http.MethodGet, ListPath, nil))
	var listed ListResponse
	if err := json.Unmarshal(list.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Commands) != 1 || listed.Commands[0].Note != "renamed" {
		t.Fatalf("listed=%+v", listed.Commands)
	}
}

func TestAddEmptyCommand(t *testing.T) {
	h := handler(t)
	rec := serve(h, postAdd(AddRequest{Command: "  "}))
	if rec.Code != 400 {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
}

func postAdd(req AddRequest) *http.Request {
	body, _ := json.Marshal(req)
	r := httptest.NewRequest(http.MethodPost, AddPath, bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func handler(t *testing.T) *Handler {
	t.Helper()
	cfg := t.TempDir()
	fixed := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	return &Handler{Store: &libcommands.Store{
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

func postUse(command string) *http.Request {
	body, _ := json.Marshal(UseRequest{Command: command})
	return httptest.NewRequest(http.MethodPost, UsePath, bytes.NewReader(body))
}
