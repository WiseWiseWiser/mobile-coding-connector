package bookmarks

import (
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

const chromeTreeFixture = `{
  "checksum": "test",
  "version": 1,
  "roots": {
    "bookmark_bar": {
      "id": "1",
      "guid": "root-bar",
      "name": "Bookmarks bar",
      "type": "folder",
      "children": [
        {
          "id": "2",
          "guid": "folder-work",
          "name": "Work",
          "type": "folder",
          "children": [
            {"id": "3", "guid": "guid-docs", "name": "Docs", "type": "url", "url": "https://docs.example.com/a"},
            {"id": "4", "guid": "guid-js", "name": "Script", "type": "url", "url": "javascript:alert(1)"}
          ]
        },
        {"id": "5", "name": "No guid", "type": "url", "url": "http://plain.example/"}
      ]
    },
    "other": {
      "id": "6",
      "guid": "root-other",
      "name": "Other bookmarks",
      "type": "folder",
      "children": [
        {"id": "7", "guid": "guid-file", "name": "File", "type": "url", "url": "file:///etc/passwd"},
        {"id": "8", "guid": "guid-blank", "name": "  ", "type": "url", "url": "https://blank.example/"}
      ]
    },
    "synced": {
      "id": "9",
      "guid": "root-synced",
      "name": "Mobile bookmarks",
      "type": "folder",
      "children": []
    }
  }
}`

const chromeAccountTreeFixture = `{
  "checksum": "account-test",
  "version": 1,
  "roots": {
    "bookmark_bar": {
      "id": "a1",
      "guid": "root-account-bar",
      "name": "Bookmarks bar",
      "type": "folder",
      "children": [
        {"id": "a2", "guid": "guid-account", "name": "Account page", "type": "url", "url": "https://account.example/"},
        {
          "id": "a3",
          "guid": "folder-work",
          "name": "Work (account)",
          "type": "folder",
          "children": [
            {"id": "a4", "guid": "guid-docs", "name": "Docs duplicate", "type": "url", "url": "https://docs.example.com/a"},
            {"id": "a5", "guid": "guid-account-in-work", "name": "Account child", "type": "url", "url": "https://account-work.example/"}
          ]
        }
      ]
    },
    "other": {
      "id": "a6",
      "guid": "root-account-other",
      "name": "Other bookmarks",
      "type": "folder",
      "children": []
    }
  }
}`

const chromeLocalStateFixture = `{
  "profile": {
    "info_cache": {
      "Default": {"name": "example.com"},
      "Profile 1": {"name": "Work"}
    }
  }
}`

func writeProfileFixture(t *testing.T, bookmarks, localState string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	profile := filepath.Join(dir, "Default")
	if err := os.MkdirAll(profile, 0o755); err != nil {
		t.Fatal(err)
	}
	bookmarksPath := filepath.Join(profile, "Bookmarks")
	if bookmarks != "" {
		if err := os.WriteFile(bookmarksPath, []byte(bookmarks), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	localStatePath := filepath.Join(dir, "Local State")
	if localState != "" {
		if err := os.WriteFile(localStatePath, []byte(localState), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return bookmarksPath, localStatePath
}

func writeAccountBookmarks(t *testing.T, bookmarksPath, content string) string {
	t.Helper()
	accountPath := filepath.Join(filepath.Dir(bookmarksPath), "AccountBookmarks")
	if err := os.WriteFile(accountPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return accountPath
}

func TestChromeReaderMapsRootsAndFiltersURLs(t *testing.T) {
	bookmarksPath, localStatePath := writeProfileFixture(t, chromeTreeFixture, chromeLocalStateFixture)
	doc, err := NewChromeReader(bookmarksPath, localStatePath).Document()
	if err != nil {
		t.Fatal(err)
	}
	if doc.Source != "chrome" || doc.Profile != "example.com" {
		t.Fatalf("source/profile = %q/%q", doc.Source, doc.Profile)
	}
	if len(doc.Roots) != 3 {
		t.Fatalf("roots = %d, want 3", len(doc.Roots))
	}
	wantRoots := []struct {
		id   string
		name string
	}{
		{"chrome:bookmark_bar", "Bookmarks Bar"},
		{"chrome:other", "Other Bookmarks"},
		{"chrome:synced", "Mobile Bookmarks"},
	}
	for i, want := range wantRoots {
		root := doc.Roots[i]
		if root.ID != want.id || root.Name != want.name || root.Type != TypeFolder {
			t.Errorf("root %d = %#v, want id %q name %q", i, root, want.id, want.name)
		}
		if root.Browser == nil || *root.Browser != "chrome" {
			t.Errorf("root %d browser = %v, want chrome", i, root.Browser)
		}
	}

	bar := doc.Roots[0]
	if len(bar.Children) != 2 {
		t.Fatalf("bar children = %d, want 2 (javascript URL skipped)", len(bar.Children))
	}
	if bar.Children[0].Name != "Work" || bar.Children[0].ID != "chrome:folder-work" {
		t.Errorf("work folder = %#v", bar.Children[0])
	}
	work := bar.Children[0]
	if len(work.Children) != 1 {
		t.Fatalf("work children = %d, want 1", len(work.Children))
	}
	if got := work.Children[0]; got.ID != "chrome:guid-docs" || got.URL != "https://docs.example.com/a" || got.Name != "Docs" {
		t.Errorf("docs = %#v", got)
	}
	if got := bar.Children[1]; got.ID != "chrome:5" || got.Name != "No guid" || got.URL != "http://plain.example/" {
		t.Errorf("guid fallback = %#v", got)
	}

	other := doc.Roots[1]
	if len(other.Children) != 1 {
		t.Fatalf("other children = %d, want 1 (file URL skipped)", len(other.Children))
	}
	if got := other.Children[0]; got.Name != "  " || got.URL != "https://blank.example/" {
		t.Errorf("blank name must be preserved for the frontend fallback: %#v", got)
	}
	if len(doc.Roots[2].Children) != 0 {
		t.Errorf("synced children = %d, want 0", len(doc.Roots[2].Children))
	}
}

func TestChromeReaderMergesLocalAndAccountBookmarks(t *testing.T) {
	bookmarksPath, localStatePath := writeProfileFixture(t, chromeTreeFixture, chromeLocalStateFixture)
	writeAccountBookmarks(t, bookmarksPath, chromeAccountTreeFixture)
	doc, err := NewChromeReader(bookmarksPath, localStatePath).Document()
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Roots) != 3 {
		t.Fatalf("roots = %d, want 3 (one merged root per key)", len(doc.Roots))
	}
	bar := doc.Roots[0]
	if len(bar.Children) != 3 {
		t.Fatalf("bar children = %d, want 3 (duplicate folder merged, duplicate URL skipped)", len(bar.Children))
	}
	if got := bar.Children[0].Name; got != "Work" {
		t.Errorf("first child = %q, want local Work first", got)
	}
	if got := bar.Children[1].Name; got != "No guid" {
		t.Errorf("second child = %q, want local No guid", got)
	}
	if got := bar.Children[2]; got.Name != "Account page" || got.URL != "https://account.example/" {
		t.Errorf("third child = %#v, want appended account page", got)
	}

	work := bar.Children[0]
	if len(work.Children) != 2 {
		t.Fatalf("work children = %d, want 2 after merging shared folder", len(work.Children))
	}
	if got := work.Children[0].Name; got != "Docs" {
		t.Errorf("work first child = %q, want local Docs", got)
	}
	if got := work.Children[1]; got.Name != "Account child" || got.URL != "https://account-work.example/" {
		t.Errorf("work second child = %#v, want merged account child", got)
	}

	if len(doc.Roots[1].Children) != 1 {
		t.Fatalf("other children = %d, want local entry only", len(doc.Roots[1].Children))
	}
	if doc.Roots[1].Children[0].URL != "https://blank.example/" {
		t.Errorf("other child = %#v", doc.Roots[1].Children[0])
	}
	if len(doc.Roots[2].Children) != 0 {
		t.Errorf("synced children = %d, want 0", len(doc.Roots[2].Children))
	}
}

func TestChromeReaderReadsAccountBookmarksWithoutLocalFile(t *testing.T) {
	bookmarksPath, localStatePath := writeProfileFixture(t, "", chromeLocalStateFixture)
	writeAccountBookmarks(t, bookmarksPath, chromeAccountTreeFixture)
	doc, err := NewChromeReader(bookmarksPath, localStatePath).Document()
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Roots) != 2 {
		t.Fatalf("roots = %d, want 2 account roots", len(doc.Roots))
	}
	if doc.Roots[0].ID != "chrome:bookmark_bar" || len(doc.Roots[0].Children) != 2 {
		t.Fatalf("bar = %#v", doc.Roots[0])
	}
	if got := doc.Roots[0].Children[0].URL; got != "https://account.example/" {
		t.Errorf("first account URL = %q", got)
	}
	if len(doc.Roots[1].Children) != 0 {
		t.Errorf("empty account other root must still be present: %#v", doc.Roots[1])
	}
}

func TestChromeReaderFallsBackToLocalWhenAccountIsMalformed(t *testing.T) {
	bookmarksPath, localStatePath := writeProfileFixture(t, chromeTreeFixture, chromeLocalStateFixture)
	writeAccountBookmarks(t, bookmarksPath, `{"roots":`)
	doc, err := NewChromeReader(bookmarksPath, localStatePath).Document()
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Roots) != 3 {
		t.Fatalf("roots = %d, want local roots", len(doc.Roots))
	}
	if len(doc.Roots[0].Children) != 2 {
		t.Fatalf("bar children = %d, want local entries only", len(doc.Roots[0].Children))
	}
}

func TestChromeReaderFallsBackToAccountWhenLocalIsMalformed(t *testing.T) {
	bookmarksPath, localStatePath := writeProfileFixture(t, `{"roots":`, chromeLocalStateFixture)
	writeAccountBookmarks(t, bookmarksPath, chromeAccountTreeFixture)
	doc, err := NewChromeReader(bookmarksPath, localStatePath).Document()
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Roots) != 2 {
		t.Fatalf("roots = %d, want account roots", len(doc.Roots))
	}
	if len(doc.Roots[0].Children) != 2 {
		t.Fatalf("bar children = %d, want account entries", len(doc.Roots[0].Children))
	}
	if got := doc.Roots[0].Children[0].URL; got != "https://account.example/" {
		t.Errorf("first account URL = %q", got)
	}
}

func TestChromeReaderBothMalformedIsAnError(t *testing.T) {
	bookmarksPath, localStatePath := writeProfileFixture(t, `{"roots":`, chromeLocalStateFixture)
	writeAccountBookmarks(t, bookmarksPath, `{"roots":`)
	_, err := NewChromeReader(bookmarksPath, localStatePath).Document()
	if !errors.Is(err, ErrChromeInvalid) {
		t.Fatalf("err = %v, want ErrChromeInvalid", err)
	}
}

func TestChromeReaderMalformedAccountDoesNotHideEmptyLocalFile(t *testing.T) {
	bookmarksPath, localStatePath := writeProfileFixture(t, "", chromeLocalStateFixture)
	writeAccountBookmarks(t, bookmarksPath, `{"roots":`)
	_, err := NewChromeReader(bookmarksPath, localStatePath).Document()
	if !errors.Is(err, ErrChromeInvalid) {
		t.Fatalf("err = %v, want ErrChromeInvalid when neither source has content", err)
	}
}

func TestChromeReaderMissingFileIsEmpty(t *testing.T) {
	bookmarksPath, localStatePath := writeProfileFixture(t, "", chromeLocalStateFixture)
	doc, err := NewChromeReader(bookmarksPath, localStatePath).Document()
	if err != nil {
		t.Fatal(err)
	}
	if doc.Source != "chrome" || doc.Profile != "example.com" {
		t.Fatalf("source/profile = %q/%q", doc.Source, doc.Profile)
	}
	if doc.Roots == nil || len(doc.Roots) != 0 {
		t.Fatalf("roots = %#v, want an empty non-nil slice", doc.Roots)
	}
}

func TestChromeReaderMissingLocalStateLeavesProfileBlank(t *testing.T) {
	bookmarksPath, _ := writeProfileFixture(t, chromeTreeFixture, "")
	doc, err := NewChromeReader(bookmarksPath, filepath.Join(filepath.Dir(filepath.Dir(bookmarksPath)), "Local State")).Document()
	if err != nil {
		t.Fatal(err)
	}
	if doc.Profile != "" {
		t.Fatalf("profile = %q, want blank", doc.Profile)
	}
	if len(doc.Roots) != 3 {
		t.Fatalf("roots = %d, want 3", len(doc.Roots))
	}
}

func TestChromeReaderMalformedFileIsAnError(t *testing.T) {
	bookmarksPath, localStatePath := writeProfileFixture(t, `{"roots":`, chromeLocalStateFixture)
	_, err := NewChromeReader(bookmarksPath, localStatePath).Document()
	if !errors.Is(err, ErrChromeInvalid) {
		t.Fatalf("err = %v, want ErrChromeInvalid", err)
	}
}

func TestChromeReaderUnreadableFileIsAnError(t *testing.T) {
	bookmarksPath, localStatePath := writeProfileFixture(t, "", chromeLocalStateFixture)
	if err := os.MkdirAll(bookmarksPath, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := NewChromeReader(bookmarksPath, localStatePath).Document()
	if !errors.Is(err, ErrChromeUnreadable) {
		t.Fatalf("err = %v, want ErrChromeUnreadable", err)
	}
}

func TestChromeReaderOnlyReadsTheFiles(t *testing.T) {
	bookmarksPath, localStatePath := writeProfileFixture(t, chromeTreeFixture, chromeLocalStateFixture)
	accountPath := writeAccountBookmarks(t, bookmarksPath, chromeAccountTreeFixture)
	beforeBookmarks, err := os.ReadFile(bookmarksPath)
	if err != nil {
		t.Fatal(err)
	}
	beforeAccount, err := os.ReadFile(accountPath)
	if err != nil {
		t.Fatal(err)
	}
	beforeLocalState, err := os.ReadFile(localStatePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewChromeReader(bookmarksPath, localStatePath).Document(); err != nil {
		t.Fatal(err)
	}
	afterBookmarks, err := os.ReadFile(bookmarksPath)
	if err != nil {
		t.Fatal(err)
	}
	afterAccount, err := os.ReadFile(accountPath)
	if err != nil {
		t.Fatal(err)
	}
	afterLocalState, err := os.ReadFile(localStatePath)
	if err != nil {
		t.Fatal(err)
	}
	if sha256.Sum256(beforeBookmarks) != sha256.Sum256(afterBookmarks) {
		t.Error("Chrome Bookmarks changed")
	}
	if sha256.Sum256(beforeAccount) != sha256.Sum256(afterAccount) {
		t.Error("Chrome AccountBookmarks changed")
	}
	if sha256.Sum256(beforeLocalState) != sha256.Sum256(afterLocalState) {
		t.Error("Chrome Local State changed")
	}
}
