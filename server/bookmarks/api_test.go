package bookmarks

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newChromeAPIFixture(t *testing.T) (*http.ServeMux, string) {
	t.Helper()
	bookmarksPath, localStatePath := writeProfileFixture(t, chromeTreeFixture, chromeLocalStateFixture)
	writeAccountBookmarks(t, bookmarksPath, chromeAccountTreeFixture)
	managerPath := filepath.Join(t.TempDir(), "bookmarks.json")
	m := NewManagerAt(managerPath)
	if _, err := m.Load(); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Add(RootID, &Node{Type: TypeURL, Name: "Local page", URL: "https://local.example/"}, nil); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	RegisterAPIWithChrome(mux, m, NewChromeReader(bookmarksPath, localStatePath))
	return mux, bookmarksPath
}

func performRequest(mux *http.ServeMux, method, target, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestChromeAPIReadsPopulatedTree(t *testing.T) {
	mux, _ := newChromeAPIFixture(t)
	rec := performRequest(mux, http.MethodGet, "/api/bookmarks?source=chrome", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	var doc Document
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Source != "chrome" || doc.Profile != "example.com" {
		t.Fatalf("source/profile = %q/%q", doc.Source, doc.Profile)
	}
	if len(doc.Roots) != 3 {
		t.Fatalf("roots = %d, want 3", len(doc.Roots))
	}
	if got := doc.Roots[0].Children[0].Children[0].URL; got != "https://docs.example.com/a" {
		t.Fatalf("docs URL = %q", got)
	}
	if len(doc.Roots[0].Children) != 3 {
		t.Fatalf("bar children = %d, want merged local + account", len(doc.Roots[0].Children))
	}
	if got := doc.Roots[0].Children[2].URL; got != "https://account.example/" {
		t.Fatalf("account URL = %q", got)
	}
}

func TestChromeAPIReadsAccountOnly(t *testing.T) {
	bookmarksPath, localStatePath := writeProfileFixture(t, "", chromeLocalStateFixture)
	writeAccountBookmarks(t, bookmarksPath, chromeAccountTreeFixture)
	mux := http.NewServeMux()
	RegisterAPIWithChrome(mux, nil, NewChromeReader(bookmarksPath, localStatePath))
	rec := performRequest(mux, http.MethodGet, "/api/bookmarks?source=chrome", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	var doc Document
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Roots) != 2 {
		t.Fatalf("roots = %d, want account roots", len(doc.Roots))
	}
	if got := doc.Roots[0].Children[0].URL; got != "https://account.example/" {
		t.Fatalf("account URL = %q", got)
	}
}

func TestChromeAPIMissingFileIsAnEmptyTree(t *testing.T) {
	bookmarksPath, localStatePath := writeProfileFixture(t, "", chromeLocalStateFixture)
	mux := http.NewServeMux()
	RegisterAPIWithChrome(mux, nil, NewChromeReader(bookmarksPath, localStatePath))
	rec := performRequest(mux, http.MethodGet, "/api/bookmarks?source=chrome", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	roots, ok := body["roots"].([]any)
	if !ok || len(roots) != 0 {
		t.Fatalf("roots = %#v, want []", body["roots"])
	}
	if body["source"] != "chrome" {
		t.Fatalf("source = %#v", body["source"])
	}
}

func TestChromeAPIMalformedFileIsA500(t *testing.T) {
	bookmarksPath, localStatePath := writeProfileFixture(t, `{"roots":`, chromeLocalStateFixture)
	mux := http.NewServeMux()
	RegisterAPIWithChrome(mux, nil, NewChromeReader(bookmarksPath, localStatePath))
	rec := performRequest(mux, http.MethodGet, "/api/bookmarks?source=chrome", "")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "could not be parsed") {
		t.Fatalf("body = %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), bookmarksPath) {
		t.Fatalf("body leaked the profile path: %s", rec.Body.String())
	}
}

func TestChromeAPIRejectsWritesWithoutTouchingTheFile(t *testing.T) {
	mux, bookmarksPath := newChromeAPIFixture(t)
	before, err := os.ReadFile(bookmarksPath)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		method string
		target string
		body   string
	}{
		{http.MethodPost, "/api/bookmarks?source=chrome", `{"parent_id":"root","type":"url","name":"X","url":"https://x.example/"}`},
		{http.MethodPatch, "/api/bookmarks?source=chrome&id=chrome:guid-docs", `{"name":"X"}`},
		{http.MethodDelete, "/api/bookmarks?source=chrome&id=chrome:guid-docs", ""},
		{http.MethodPost, "/api/bookmarks/move?source=chrome", `{"id":"chrome:guid-docs","parent_id":"chrome:bookmark_bar"}`},
	}
	for _, tt := range tests {
		rec := performRequest(mux, tt.method, tt.target, tt.body)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s %s status = %d, body %s", tt.method, tt.target, rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "read-only") {
			t.Errorf("%s %s body = %s", tt.method, tt.target, rec.Body.String())
		}
	}
	after, err := os.ReadFile(bookmarksPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("Chrome Bookmarks changed after rejected writes")
	}
}

func TestChromeAPIRejectsUnknownSource(t *testing.T) {
	mux, _ := newChromeAPIFixture(t)
	rec := performRequest(mux, http.MethodGet, "/api/bookmarks?source=brave", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "source must be empty or chrome") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestLocalAPIShapeAndWritesAreUnchanged(t *testing.T) {
	mux, _ := newChromeAPIFixture(t)
	rec := performRequest(mux, http.MethodGet, "/api/bookmarks", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	var doc Document
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Source != "" || doc.Profile != "" {
		t.Fatalf("local source/profile leaked: %q/%q", doc.Source, doc.Profile)
	}
	if len(doc.Roots) != 1 || len(doc.Roots[0].Children) != 1 {
		t.Fatalf("local roots = %#v", doc.Roots)
	}

	rec = performRequest(mux, http.MethodPost, "/api/bookmarks", `{"parent_id":"root","type":"url","name":"Second","url":"https://second.example/"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("local POST status = %d, body %s", rec.Code, rec.Body.String())
	}
	rec = performRequest(mux, http.MethodGet, "/api/bookmarks", "")
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Roots[0].Children) != 2 {
		t.Fatalf("local children = %d, want 2", len(doc.Roots[0].Children))
	}
}

func TestChromeReadWorksWithoutALocalManager(t *testing.T) {
	bookmarksPath, localStatePath := writeProfileFixture(t, chromeTreeFixture, chromeLocalStateFixture)
	mux := http.NewServeMux()
	RegisterAPIWithChrome(mux, nil, NewChromeReader(bookmarksPath, localStatePath))
	rec := performRequest(mux, http.MethodGet, "/api/bookmarks", "")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("local status = %d, want 500", rec.Code)
	}
	rec = performRequest(mux, http.MethodGet, "/api/bookmarks?source=chrome", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("chrome status = %d, body %s", rec.Code, rec.Body.String())
	}
}
