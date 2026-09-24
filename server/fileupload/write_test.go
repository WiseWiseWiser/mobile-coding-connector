package fileupload

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"

	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func md5Of(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

type writeRequest struct {
	path        string
	expectedMD5 string
	content     string
	// omitExpected drops the expected_md5 form field entirely.
	omitExpected bool
}

func postWrite(t *testing.T, req writeRequest) *httptest.ResponseRecorder {
	t.Helper()

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if req.path != "" {
		if err := mw.WriteField("path", req.path); err != nil {
			t.Fatal(err)
		}
	}
	if !req.omitExpected {
		if err := mw.WriteField("expected_md5", req.expectedMD5); err != nil {
			t.Fatal(err)
		}
	}
	part, err := mw.CreateFormFile("file", "content")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte(req.content)); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}

	httpReq := httptest.NewRequest(http.MethodPost, WritePath, &body)
	httpReq.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	handleWrite(rec, httpReq)
	return rec
}

func decodeWriteResponse(t *testing.T, rec *httptest.ResponseRecorder) WriteResponse {
	t.Helper()
	var out WriteResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode write response %q: %v", rec.Body.String(), err)
	}
	return out
}

func decodeConflict(t *testing.T, rec *httptest.ResponseRecorder) WriteConflictResponse {
	t.Helper()
	var out WriteConflictResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode conflict response %q: %v", rec.Body.String(), err)
	}
	return out
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func TestWriteReplacesFileWhenMD5Matches(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(target, []byte("old\n"), 0644); err != nil {
		t.Fatal(err)
	}

	rec := postWrite(t, writeRequest{path: target, expectedMD5: md5Of("old\n"), content: "new\n"})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	got := decodeWriteResponse(t, rec)
	if got.MD5 != md5Of("new\n") {
		t.Fatalf("md5 = %q, want %q", got.MD5, md5Of("new\n"))
	}
	if got.Created {
		t.Fatal("created = true, want false")
	}
	if got.Path != target || got.ResolvedPath != target {
		t.Fatalf("paths = %q / %q, want %q", got.Path, got.ResolvedPath, target)
	}
	if readFile(t, target) != "new\n" {
		t.Fatalf("content = %q", readFile(t, target))
	}
	assertNoTempFiles(t, dir)
}

func TestWriteRejectsChangedFileAndLeavesItIntact(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(target, []byte("changed elsewhere\n"), 0644); err != nil {
		t.Fatal(err)
	}

	rec := postWrite(t, writeRequest{path: target, expectedMD5: md5Of("old\n"), content: "mine\n"})
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	conflict := decodeConflict(t, rec)
	if conflict.ExpectedMD5 != md5Of("old\n") || conflict.CurrentMD5 != md5Of("changed elsewhere\n") {
		t.Fatalf("hashes = %q / %q", conflict.ExpectedMD5, conflict.CurrentMD5)
	}
	if !conflict.CurrentExists {
		t.Fatal("current_exists = false, want true")
	}
	if conflict.CurrentSize != int64(len("changed elsewhere\n")) {
		t.Fatalf("current_size = %d", conflict.CurrentSize)
	}
	if conflict.CurrentMod == "" {
		t.Fatal("current_mod_time empty")
	}
	if readFile(t, target) != "changed elsewhere\n" {
		t.Fatalf("file was modified on conflict: %q", readFile(t, target))
	}
	assertNoTempFiles(t, dir)
}

func TestWriteCreatesFileWhenExpectedEmptyMD5(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "nested", "new.txt")

	rec := postWrite(t, writeRequest{path: target, expectedMD5: EmptyMD5, content: "hello\n"})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	got := decodeWriteResponse(t, rec)
	if !got.Created {
		t.Fatal("created = false, want true")
	}
	if readFile(t, target) != "hello\n" {
		t.Fatalf("content = %q", readFile(t, target))
	}
}

func TestWriteRejectsCreatedFileWhenExpectedAbsent(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "new.txt")
	if err := os.WriteFile(target, []byte("raced\n"), 0644); err != nil {
		t.Fatal(err)
	}

	rec := postWrite(t, writeRequest{path: target, expectedMD5: EmptyMD5, content: "mine\n"})
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if readFile(t, target) != "raced\n" {
		t.Fatalf("file was modified on conflict: %q", readFile(t, target))
	}
}

func TestWriteTreatsMissingFileAsEmptyContent(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "gone.txt")

	rec := postWrite(t, writeRequest{path: target, expectedMD5: md5Of("old\n"), content: "mine\n"})
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	conflict := decodeConflict(t, rec)
	if conflict.CurrentExists {
		t.Fatal("current_exists = true, want false")
	}
	if conflict.CurrentMD5 != EmptyMD5 {
		t.Fatalf("current_md5 = %q, want %q", conflict.CurrentMD5, EmptyMD5)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("file should not exist, stat err = %v", err)
	}
}

func TestWriteWithoutPreconditionAlwaysWrites(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(target, []byte("old\n"), 0644); err != nil {
		t.Fatal(err)
	}

	rec := postWrite(t, writeRequest{path: target, content: "forced\n", omitExpected: true})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if readFile(t, target) != "forced\n" {
		t.Fatalf("content = %q", readFile(t, target))
	}
}

func TestWritePreservesMode(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "run.sh")
	if err := os.WriteFile(target, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}

	rec := postWrite(t, writeRequest{path: target, expectedMD5: md5Of("#!/bin/sh\n"), content: "#!/bin/sh\necho hi\n"})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0755 {
		t.Fatalf("mode = %v, want 0755", info.Mode().Perm())
	}
}

func TestWriteThroughSymlinkKeepsLink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "sites-available", "app.conf")
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("old\n"), 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "app.conf")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	rec := postWrite(t, writeRequest{path: link, expectedMD5: md5Of("old\n"), content: "new\n"})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	got := decodeWriteResponse(t, rec)
	if got.ResolvedPath != target {
		t.Fatalf("resolved_path = %q, want %q", got.ResolvedPath, target)
	}
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("symlink was replaced by a regular file")
	}
	if readFile(t, target) != "new\n" {
		t.Fatalf("target content = %q", readFile(t, target))
	}
}

func TestWriteCreatesDanglingSymlinkTarget(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "missing-target.conf")
	link := filepath.Join(dir, "app.conf")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	rec := postWrite(t, writeRequest{path: link, expectedMD5: EmptyMD5, content: "new\n"})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("symlink was replaced by a regular file")
	}
	if readFile(t, target) != "new\n" {
		t.Fatalf("target content = %q", readFile(t, target))
	}
}

func TestWriteRejectsDirectory(t *testing.T) {
	dir := t.TempDir()

	rec := postWrite(t, writeRequest{path: dir, expectedMD5: EmptyMD5, content: "x"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "directory") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestWriteRequiresPath(t *testing.T) {
	rec := postWrite(t, writeRequest{expectedMD5: EmptyMD5, content: "x"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestWriteRejectsNonPost(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, WritePath, nil)
	rec := httptest.NewRecorder()
	handleWrite(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestResolveWritePathFollowsParentSymlink(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real")
	if err := os.MkdirAll(real, 0755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}

	got, err := resolveWritePath(filepath.Join(link, "sub", "new.txt"), 0)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(real, "sub", "new.txt")
	if got != want {
		t.Fatalf("resolved = %q, want %q", got, want)
	}
}

func TestResolveWritePathRejectsSymlinkLoop(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	b := filepath.Join(dir, "b")
	if err := os.Symlink(b, a); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(a, b); err != nil {
		t.Fatal(err)
	}

	if _, err := resolveWritePath(a, 0); err == nil {
		t.Fatal("expected symlink loop error")
	}
}

// assertNoTempFiles guards the atomic-write path: no .remote-agent-write-*
// leftovers may remain after a request, successful or not.
func assertNoTempFiles(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".remote-agent-write-") {
			t.Fatalf("leftover temp file %s", filepath.Join(dir, e.Name()))
		}
	}
}

func TestWriteViaRegisteredMux(t *testing.T) {
	home := t.TempDir()
	mux := http.NewServeMux()
	RegisterAPIForHome(mux, home)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	target := filepath.Join(home, "mux.txt")

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("path", target)
	_ = mw.WriteField("expected_md5", EmptyMD5)
	part, _ := mw.CreateFormFile("file", "content")
	_, _ = part.Write([]byte("via mux\n"))
	_ = mw.Close()

	resp, err := http.Post(srv.URL+WritePath, mw.FormDataContentType(), &body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if readFile(t, target) != "via mux\n" {
		t.Fatalf("content = %q", readFile(t, target))
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("content-type = %q", ct)
	}
}
