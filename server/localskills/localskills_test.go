package localskills

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xhd2015/dot-pkgs/go-pkgs/fuzzy"
	"github.com/xhd2015/my/libskills"
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
	if out.Skills == nil || len(out.Skills) != 0 {
		t.Fatalf("skills=%v", out.Skills)
	}
	if out.MissingRoots == nil || len(out.MissingRoots) != 0 {
		t.Fatalf("missing=%v", out.MissingRoots)
	}
}

func TestListRankedAfterUse(t *testing.T) {
	h := handler(t)
	root := filepath.Join(h.Store.ConfigDir, "root")
	writeSkill(t, filepath.Join(root, "alpha"))
	writeSkill(t, filepath.Join(root, "beta"))
	if err := libskills.SaveFile(h.Store.ConfigDir, &libskills.File{
		SkillDirs: []libskills.SkillDirEntry{{Path: root, AddedAt: "2026-01-01T00:00:00Z"}},
	}); err != nil {
		t.Fatal(err)
	}

	beta := filepath.Join(root, "beta", "SKILL.md")
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
	if len(out.Skills) != 2 {
		t.Fatalf("len=%d", len(out.Skills))
	}
	if out.Skills[0].Name != "beta" || out.Skills[0].UseCount != 2 {
		t.Fatalf("first=%+v", out.Skills[0])
	}
	if out.Skills[1].Name != "alpha" || out.Skills[1].UseCount != 0 {
		t.Fatalf("second=%+v", out.Skills[1])
	}
}

func TestListQueryAND(t *testing.T) {
	h := handler(t)
	root := filepath.Join(h.Store.ConfigDir, "root")
	writeSkill(t, filepath.Join(root, "aid-user-do-human-verifications"))
	writeSkill(t, filepath.Join(root, "followup"))
	if err := libskills.SaveFile(h.Store.ConfigDir, &libskills.File{
		SkillDirs: []libskills.SkillDirEntry{{Path: root, AddedAt: "2026-01-01T00:00:00Z"}},
	}); err != nil {
		t.Fatal(err)
	}
	rec := serve(h, httptest.NewRequest(http.MethodGet, ListPath+"?q=aid+user", nil))
	if rec.Code != 200 {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	var out ListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Skills) != 1 || out.Skills[0].Name != "aid-user-do-human-verifications" {
		t.Fatalf("skills=%+v", out.Skills)
	}
	if !spanHasMatched(out.Skills[0].TitleSpans, "aid") || !spanHasMatched(out.Skills[0].TitleSpans, "user") {
		t.Fatalf("title_spans=%+v", out.Skills[0].TitleSpans)
	}
	joined := ""
	for _, s := range out.Skills[0].TitleSpans {
		joined += s.Text
	}
	if joined != "aid-user-do-human-verifications" {
		t.Fatalf("join=%q", joined)
	}
	raw := rec.Body.String()
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"text"`)) {
		t.Fatalf("span JSON must use lowercase text/matched keys:\n%s", raw)
	}
	if bytes.Contains(rec.Body.Bytes(), []byte(`"Text"`)) {
		t.Fatalf("span JSON used Go field names:\n%s", raw)
	}
}

func TestListQueryMiss(t *testing.T) {
	h := handler(t)
	root := filepath.Join(h.Store.ConfigDir, "root")
	writeSkill(t, filepath.Join(root, "followup"))
	if err := libskills.SaveFile(h.Store.ConfigDir, &libskills.File{
		SkillDirs: []libskills.SkillDirEntry{{Path: root, AddedAt: "2026-01-01T00:00:00Z"}},
	}); err != nil {
		t.Fatal(err)
	}
	rec := serve(h, httptest.NewRequest(http.MethodGet, ListPath+"?q=zzz", nil))
	if rec.Code != 200 {
		t.Fatalf("code=%d", rec.Code)
	}
	var out ListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Skills) != 0 {
		t.Fatalf("skills=%+v", out.Skills)
	}
}

func spanHasMatched(spans []fuzzy.Span, want string) bool {
	for _, s := range spans {
		if s.Matched && s.Text == want {
			return true
		}
	}
	return false
}

func TestUseUnknown(t *testing.T) {
	h := handler(t)
	rec := serve(h, postUse("/no/such/SKILL.md"))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestUseMissingPath(t *testing.T) {
	h := handler(t)
	rec := serve(h, httptest.NewRequest(http.MethodPost, UsePath, bytes.NewReader([]byte(`{}`))))
	if rec.Code != http.StatusNotFound && rec.Code != http.StatusBadRequest {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
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
	fixed := time.Date(2026, 8, 19, 15, 0, 0, 0, time.UTC)
	return &Handler{Store: &libskills.Store{
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

func writeSkill(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# s\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}
