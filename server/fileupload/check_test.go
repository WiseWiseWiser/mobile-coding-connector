package fileupload

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func postCheck(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/files/check", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleCheck(rec, req)
	return rec
}

func decodeCheck(t *testing.T, rec *httptest.ResponseRecorder) FileInfo {
	t.Helper()
	var out FileInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode check response %q: %v", rec.Body.String(), err)
	}
	return out
}

// checkRawPayload decodes the response into a map so a test can assert that a
// field is absent rather than empty.
func checkRawPayload(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var raw map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode check response %q: %v", rec.Body.String(), err)
	}
	return raw
}

func TestCheckReturnsDigestWhenRequested(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.md")
	if err := os.WriteFile(path, []byte("old\n"), 0644); err != nil {
		t.Fatal(err)
	}

	rec := postCheck(t, `{"path":`+jsonString(path)+`,"md5":true}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	info := decodeCheck(t, rec)
	if !info.Exists {
		t.Fatal("exists = false")
	}
	if info.MD5 != md5Of("old\n") {
		t.Fatalf("md5 = %q, want %q", info.MD5, md5Of("old\n"))
	}
	if info.Size != 4 {
		t.Fatalf("size = %d, want 4", info.Size)
	}
}

func TestCheckOmitsDigestByDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.md")
	if err := os.WriteFile(path, []byte("old\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Plain check (what download and other callers use): no digest, no hashing.
	rec := postCheck(t, `{"path":`+jsonString(path)+`}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if _, ok := checkRawPayload(t, rec)["md5"]; ok {
		t.Fatalf("md5 must be omitted unless requested: %s", rec.Body.String())
	}
}

func TestCheckDigestForMissingPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "absent.md")

	rec := postCheck(t, `{"path":`+jsonString(path)+`,"md5":true}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	info := decodeCheck(t, rec)
	if info.Exists {
		t.Fatal("exists = true, want false")
	}
	if info.MD5 != "" {
		t.Fatalf("md5 = %q, want empty for a missing path", info.MD5)
	}
}

func TestCheckDigestForDirectory(t *testing.T) {
	dir := t.TempDir()

	rec := postCheck(t, `{"path":`+jsonString(dir)+`,"md5":true}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	info := decodeCheck(t, rec)
	if !info.IsDir {
		t.Fatal("is_dir = false, want true")
	}
	if info.MD5 != "" {
		t.Fatalf("md5 = %q, want empty for a directory", info.MD5)
	}
}

func TestCheckDigestMatchesFileContentChange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.md")
	if err := os.WriteFile(path, []byte("old\n"), 0644); err != nil {
		t.Fatal(err)
	}

	before := decodeCheck(t, postCheck(t, `{"path":`+jsonString(path)+`,"md5":true}`))
	if err := os.WriteFile(path, []byte("new\n"), 0644); err != nil {
		t.Fatal(err)
	}
	after := decodeCheck(t, postCheck(t, `{"path":`+jsonString(path)+`,"md5":true}`))

	if before.MD5 == after.MD5 {
		t.Fatalf("digest did not change with content: %q", before.MD5)
	}
	if after.MD5 != md5Of("new\n") {
		t.Fatalf("md5 = %q, want %q", after.MD5, md5Of("new\n"))
	}
}

// jsonString encodes s as a JSON string literal for hand-built request bodies.
func jsonString(s string) string {
	data, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(data)
}
