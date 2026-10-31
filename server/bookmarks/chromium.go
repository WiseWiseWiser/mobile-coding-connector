package bookmarks

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	// ErrChromeUnreadable means a Chrome bookmark file exists but could not be
	// opened. It is deliberately distinct from a missing file: missing is a
	// valid empty source, unreadable is an error.
	ErrChromeUnreadable = errors.New("Chrome bookmarks are not readable")
	// ErrChromeInvalid means a Chrome bookmark file was read but is not the
	// JSON tree Chrome writes.
	ErrChromeInvalid = errors.New("Chrome bookmarks are invalid")
)

var chromeBrowser = "chrome"

// ChromeReader reads the normal Chrome profile's bookmark tree. Chrome keeps
// device-local bookmarks in Bookmarks and signed-in account bookmarks in
// AccountBookmarks; both are needed to match what Chrome shows. It only ever
// reads: Chrome owns both files and their checksums, so writing them behind
// Chrome's back would be unsafe.
type ChromeReader struct {
	bookmarksPath        string
	accountBookmarksPath string
	localStatePath       string
}

// NewChromeReader reads bookmarksPath, the AccountBookmarks file beside it,
// and the Local State file beside its profile. Tests inject temporary paths;
// production uses DefaultChromeReader.
func NewChromeReader(bookmarksPath, localStatePath string) *ChromeReader {
	return &ChromeReader{
		bookmarksPath:        bookmarksPath,
		accountBookmarksPath: filepath.Join(filepath.Dir(bookmarksPath), "AccountBookmarks"),
		localStatePath:       localStatePath,
	}
}

// DefaultChromeReader points at the normal Chrome profile for this OS.
func DefaultChromeReader() (*ChromeReader, error) {
	profileDir, err := defaultChromeProfileDir()
	if err != nil {
		return nil, err
	}
	return NewChromeReader(
		filepath.Join(profileDir, "Bookmarks"),
		filepath.Join(filepath.Dir(profileDir), "Local State"),
	), nil
}

func defaultChromeProfileDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Google", "Chrome", "Default"), nil
	case "windows":
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			return "", errors.New("LOCALAPPDATA is not set")
		}
		return filepath.Join(base, "Google", "Chrome", "User Data", "Default"), nil
	default:
		return filepath.Join(home, ".config", "google-chrome", "Default"), nil
	}
}

// Document returns Chrome's device-local and signed-in account trees merged
// into the same shape as the app-local document. A missing file is an empty
// source, which is normal for a fresh profile or a signed-out one. When only
// one file is usable, that half is still returned rather than hiding every
// bookmark behind one corrupt file. Both files are reread on every call, so
// Chrome changes show up on the next refresh without a cache or watcher.
func (r *ChromeReader) Document() (*Document, error) {
	doc := &Document{
		Version: DocVersion,
		Roots:   []*Node{},
		Source:  "chrome",
		Profile: r.profileName(),
	}

	local, localPresent, localErr := readChromeBookmarkFile(r.bookmarksPath)
	account, accountPresent, accountErr := readChromeBookmarkFile(r.accountBookmarksPath)
	if localErr != nil || accountErr != nil {
		switch {
		case localErr != nil && accountErr != nil:
			return nil, localErr
		case localErr != nil:
			if !accountPresent || !chromeHasContent(account) {
				return nil, localErr
			}
			local = nil
		case accountErr != nil:
			if !localPresent || !chromeHasContent(local) {
				return nil, accountErr
			}
			account = nil
		}
	}

	doc.Roots = mergeChromeRoots(local, account)
	return doc, nil
}

func readChromeBookmarkFile(path string) (*chromeBookmarkFile, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("%w: %v", ErrChromeUnreadable, err)
	}
	var file chromeBookmarkFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, false, fmt.Errorf("%w: %v", ErrChromeInvalid, err)
	}
	return &file, true, nil
}

func chromeHasContent(file *chromeBookmarkFile) bool {
	if file == nil {
		return false
	}
	for _, root := range []*chromeNode{file.Roots.BookmarkBar, file.Roots.Other, file.Roots.Synced} {
		if root != nil && len(root.Children) > 0 {
			return true
		}
	}
	return false
}

type chromeBookmarkFile struct {
	Roots struct {
		BookmarkBar *chromeNode `json:"bookmark_bar"`
		Other       *chromeNode `json:"other"`
		Synced      *chromeNode `json:"synced"`
	} `json:"roots"`
}

type chromeNode struct {
	ID       string        `json:"id"`
	GUID     string        `json:"guid"`
	Name     string        `json:"name"`
	Type     string        `json:"type"`
	URL      string        `json:"url"`
	Children []*chromeNode `json:"children"`
}

func mergeChromeRoots(local, account *chromeBookmarkFile) []*Node {
	roots := []struct {
		key     string
		name    string
		local   *chromeNode
		account *chromeNode
	}{
		{"bookmark_bar", "Bookmarks Bar", chromeRootNode(local, "bookmark_bar"), chromeRootNode(account, "bookmark_bar")},
		{"other", "Other Bookmarks", chromeRootNode(local, "other"), chromeRootNode(account, "other")},
		{"synced", "Mobile Bookmarks", chromeRootNode(local, "synced"), chromeRootNode(account, "synced")},
	}

	out := make([]*Node, 0, len(roots))
	for _, root := range roots {
		if root.local == nil && root.account == nil {
			continue
		}
		var children []*chromeNode
		if root.local != nil {
			children = root.local.Children
		}
		if root.account != nil {
			children = mergeChromeNodes(children, root.account.Children)
		}
		out = append(out, &Node{
			Type:     TypeFolder,
			ID:       "chrome:" + root.key,
			Name:     root.name,
			Browser:  &chromeBrowser,
			Children: chromeChildren(children),
		})
	}
	return out
}

func chromeRootNode(file *chromeBookmarkFile, key string) *chromeNode {
	if file == nil {
		return nil
	}
	switch key {
	case "bookmark_bar":
		return file.Roots.BookmarkBar
	case "other":
		return file.Roots.Other
	case "synced":
		return file.Roots.Synced
	default:
		return nil
	}
}

// mergeChromeNodes keeps the local tree first, then appends account entries
// that are not already present. GUIDs are stable across Chrome's two stores, so
// a shared GUID means the same node: merge folders recursively and skip the
// duplicate rather than showing it twice. Nodes without a GUID are never merged
// by their numeric ID, because the two files number nodes independently.
// Different GUIDs with the same URL are preserved because Chrome allows
// intentional duplicates.
func mergeChromeNodes(local, account []*chromeNode) []*chromeNode {
	out := make([]*chromeNode, 0, len(local)+len(account))
	seen := make(map[string]*chromeNode, len(local))
	for _, node := range local {
		if node == nil {
			continue
		}
		out = append(out, node)
		if key := chromeMergeKey(node); key != "" {
			seen[key] = node
		}
	}
	for _, node := range account {
		if node == nil {
			continue
		}
		if key := chromeMergeKey(node); key != "" {
			if existing, ok := seen[key]; ok {
				if existing.Type == TypeFolder && node.Type == TypeFolder {
					existing.Children = mergeChromeNodes(existing.Children, node.Children)
				}
				continue
			}
			seen[key] = node
		}
		out = append(out, node)
	}
	return out
}

func chromeMergeKey(node *chromeNode) string {
	if node.GUID == "" {
		return ""
	}
	return "guid:" + node.GUID
}

func chromeChildren(nodes []*chromeNode) []*Node {
	out := make([]*Node, 0, len(nodes))
	for _, node := range nodes {
		if node == nil {
			continue
		}
		switch node.Type {
		case TypeFolder:
			out = append(out, &Node{
				Type:     TypeFolder,
				ID:       chromeNodeID(node),
				Name:     node.Name,
				Browser:  &chromeBrowser,
				Children: chromeChildren(node.Children),
			})
		case TypeURL:
			rawURL := strings.TrimSpace(node.URL)
			if !isChromeHTTPURL(rawURL) {
				continue
			}
			out = append(out, &Node{
				Type:    TypeURL,
				ID:      chromeNodeID(node),
				Name:    node.Name,
				URL:     rawURL,
				Browser: &chromeBrowser,
			})
		}
	}
	return out
}

func chromeNodeID(node *chromeNode) string {
	if node.GUID != "" {
		return "chrome:" + node.GUID
	}
	return "chrome:" + node.ID
}

func isChromeHTTPURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return false
	}
	scheme := strings.ToLower(parsed.Scheme)
	return scheme == "http" || scheme == "https"
}

func (r *ChromeReader) profileName() string {
	data, err := os.ReadFile(r.localStatePath)
	if err != nil {
		return ""
	}
	var state struct {
		Profile struct {
			InfoCache map[string]struct {
				Name string `json:"name"`
			} `json:"info_cache"`
		} `json:"profile"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return ""
	}
	key := filepath.Base(filepath.Dir(r.bookmarksPath))
	return strings.TrimSpace(state.Profile.InfoCache[key].Name)
}
