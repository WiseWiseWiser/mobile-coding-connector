package bookmarks

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	RootID      = "root"
	RootName    = "Bookmarks"
	TypeFolder  = "folder"
	TypeURL     = "url"
	DocVersion  = 1
)

var (
	ErrNotFound      = errors.New("bookmark not found")
	ErrCannotDeleteRoot = errors.New("cannot delete root folder")
	ErrInvalidName   = errors.New("name is required")
	ErrInvalidURL    = errors.New("url must be an absolute http or https URL")
	ErrInvalidType   = errors.New("type must be folder or url")
	ErrInvalidBrowser = errors.New("browser must be empty, default, chrome, firefox, or opera")
	ErrParentNotFolder = errors.New("parent is not a folder")
	ErrBadParent     = errors.New("parent not found")
)

// Manager owns a bookmarks.json file and in-memory tree.
type Manager struct {
	path string
	mu   sync.Mutex
	doc  *Document
}

// NewManagerAt creates a manager for the given bookmarks.json path.
func NewManagerAt(path string) *Manager {
	return &Manager{path: path}
}

// DefaultPath returns ~/.ai-critic/bookmarks.json.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ai-critic", "bookmarks.json"), nil
}

// NewDefaultManager loads from the default path.
func NewDefaultManager() (*Manager, error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, err
	}
	m := NewManagerAt(path)
	if _, err := m.Load(); err != nil {
		return nil, err
	}
	return m, nil
}

// Document returns the current in-memory document (after Load/mutations).
func (m *Manager) Document() *Document {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.doc
}

// Load reads bookmarks.json; missing/empty file → default root.
func (m *Manager) Load() (*Document, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.path)
	if err != nil {
		if os.IsNotExist(err) {
			m.doc = defaultDocument()
			return m.doc, nil
		}
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		m.doc = defaultDocument()
		return m.doc, nil
	}
	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse bookmarks.json: %w", err)
	}
	if doc.Version == 0 {
		doc.Version = DocVersion
	}
	if len(doc.Roots) == 0 {
		doc.Roots = []*Node{defaultRoot()}
	}
	m.doc = &doc
	return m.doc, nil
}

func defaultDocument() *Document {
	return &Document{
		Version: DocVersion,
		Roots:   []*Node{defaultRoot()},
	}
}

func defaultRoot() *Node {
	return &Node{
		Type:     TypeFolder,
		ID:       RootID,
		Name:     RootName,
		Children: []*Node{},
	}
}

func (m *Manager) saveLocked() error {
	if m.doc == nil {
		m.doc = defaultDocument()
	}
	if err := os.MkdirAll(filepath.Dir(m.path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m.doc, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := m.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, m.path)
}

// Add inserts a node under parentID. index nil appends.
func (m *Manager) Add(parentID string, n *Node, index *int) (*Node, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.doc == nil {
		m.doc = defaultDocument()
	}
	if parentID == "" {
		parentID = RootID
	}
	if n == nil {
		return nil, fmt.Errorf("node is required")
	}
	node, err := normalizeNewNode(n)
	if err != nil {
		return nil, err
	}
	parent := findNodeLocked(m.doc.Roots, parentID)
	if parent == nil {
		return nil, ErrBadParent
	}
	if parent.Type != TypeFolder {
		return nil, ErrParentNotFolder
	}
	if findNodeLocked(m.doc.Roots, node.ID) != nil {
		return nil, fmt.Errorf("id already exists: %s", node.ID)
	}
	if parent.Children == nil {
		parent.Children = []*Node{}
	}
	if index == nil || *index < 0 || *index >= len(parent.Children) {
		parent.Children = append(parent.Children, node)
	} else {
		i := *index
		parent.Children = append(parent.Children, nil)
		copy(parent.Children[i+1:], parent.Children[i:])
		parent.Children[i] = node
	}
	if err := m.saveLocked(); err != nil {
		return nil, err
	}
	return node, nil
}

// Update patches an existing node by id.
func (m *Manager) Update(id string, opts UpdateOpts) (*Node, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.doc == nil {
		m.doc = defaultDocument()
	}
	n := findNodeLocked(m.doc.Roots, id)
	if n == nil {
		return nil, ErrNotFound
	}
	if opts.Name != nil {
		name := strings.TrimSpace(*opts.Name)
		if name == "" {
			return nil, ErrInvalidName
		}
		n.Name = name
	}
	if opts.URL != nil {
		u := strings.TrimSpace(*opts.URL)
		if n.Type == TypeURL {
			if err := validateURL(u); err != nil {
				return nil, err
			}
			n.URL = u
		} else if u != "" {
			return nil, fmt.Errorf("folder cannot have url")
		}
	}
	if opts.ClearBrowser {
		n.Browser = nil
	} else if opts.Browser != nil {
		b := strings.TrimSpace(*opts.Browser)
		if b == "" {
			n.Browser = nil
		} else {
			if err := validateBrowser(b); err != nil {
				return nil, err
			}
			n.Browser = &b
		}
	}
	if err := m.saveLocked(); err != nil {
		return nil, err
	}
	return n, nil
}

// Delete removes a node by id. Root cannot be deleted. Folders are recursive.
func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.doc == nil {
		m.doc = defaultDocument()
	}
	if id == RootID {
		return ErrCannotDeleteRoot
	}
	if findNodeLocked(m.doc.Roots, id) == nil {
		return ErrNotFound
	}
	if !removeNodeLocked(&m.doc.Roots, id) {
		return ErrNotFound
	}
	return m.saveLocked()
}

// Move reparents id under parentID with optional index.
func (m *Manager) Move(id, parentID string, index *int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.doc == nil {
		m.doc = defaultDocument()
	}
	if id == RootID {
		return fmt.Errorf("cannot move root folder")
	}
	if parentID == "" {
		parentID = RootID
	}
	node := findNodeLocked(m.doc.Roots, id)
	if node == nil {
		return ErrNotFound
	}
	// Prevent moving a folder into itself or a descendant.
	if node.Type == TypeFolder && (id == parentID || findNodeLocked(node.Children, parentID) != nil) {
		return fmt.Errorf("cannot move folder into itself or a descendant")
	}
	parent := findNodeLocked(m.doc.Roots, parentID)
	if parent == nil {
		return ErrBadParent
	}
	if parent.Type != TypeFolder {
		return ErrParentNotFolder
	}
	// Detach copy
	cloned := cloneNode(node)
	if !removeNodeLocked(&m.doc.Roots, id) {
		return ErrNotFound
	}
	// Re-find parent after remove (pointers may still be valid if not removed parent)
	parent = findNodeLocked(m.doc.Roots, parentID)
	if parent == nil {
		return ErrBadParent
	}
	if parent.Children == nil {
		parent.Children = []*Node{}
	}
	if index == nil || *index < 0 || *index >= len(parent.Children) {
		parent.Children = append(parent.Children, cloned)
	} else {
		i := *index
		parent.Children = append(parent.Children, nil)
		copy(parent.Children[i+1:], parent.Children[i:])
		parent.Children[i] = cloned
	}
	return m.saveLocked()
}

func cloneNode(n *Node) *Node {
	if n == nil {
		return nil
	}
	out := &Node{
		Type: n.Type,
		ID:   n.ID,
		Name: n.Name,
		URL:  n.URL,
	}
	if n.Browser != nil {
		b := *n.Browser
		out.Browser = &b
	}
	if len(n.Children) > 0 {
		out.Children = make([]*Node, len(n.Children))
		for i, c := range n.Children {
			out.Children[i] = cloneNode(c)
		}
	} else if n.Type == TypeFolder {
		out.Children = []*Node{}
	}
	return out
}

func normalizeNewNode(n *Node) (*Node, error) {
	typ := strings.TrimSpace(n.Type)
	if typ == "" {
		if n.URL != "" {
			typ = TypeURL
		} else {
			typ = TypeFolder
		}
	}
	if typ != TypeFolder && typ != TypeURL {
		return nil, ErrInvalidType
	}
	name := strings.TrimSpace(n.Name)
	if name == "" {
		return nil, ErrInvalidName
	}
	id := strings.TrimSpace(n.ID)
	if id == "" {
		id = generateID(typ)
	}
	out := &Node{
		Type: typ,
		ID:   id,
		Name: name,
	}
	if typ == TypeURL {
		u := strings.TrimSpace(n.URL)
		if err := validateURL(u); err != nil {
			return nil, err
		}
		out.URL = u
	} else {
		out.Children = []*Node{}
		if n.Children != nil {
			// Do not accept nested children on create for simplicity
			out.Children = []*Node{}
		}
	}
	if n.Browser != nil {
		b := strings.TrimSpace(*n.Browser)
		if b == "" {
			out.Browser = nil
		} else {
			if err := validateBrowser(b); err != nil {
				return nil, err
			}
			out.Browser = &b
		}
	}
	return out, nil
}

func validateURL(raw string) error {
	if raw == "" {
		return ErrInvalidURL
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ErrInvalidURL
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return ErrInvalidURL
	}
	return nil
}

func validateBrowser(b string) error {
	switch strings.ToLower(b) {
	case "default", "chrome", "firefox", "opera":
		return nil
	default:
		return ErrInvalidBrowser
	}
}

func generateID(typ string) string {
	var buf [8]byte
	_, _ = rand.Read(buf[:])
	prefix := "bm_"
	if typ == TypeFolder {
		prefix = "fld_"
	}
	return prefix + hex.EncodeToString(buf[:])
}

func findNodeLocked(nodes []*Node, id string) *Node {
	for _, n := range nodes {
		if n == nil {
			continue
		}
		if n.ID == id {
			return n
		}
		if found := findNodeLocked(n.Children, id); found != nil {
			return found
		}
	}
	return nil
}

func removeNodeLocked(nodes *[]*Node, id string) bool {
	if nodes == nil {
		return false
	}
	for i, n := range *nodes {
		if n == nil {
			continue
		}
		if n.ID == id {
			*nodes = append((*nodes)[:i], (*nodes)[i+1:]...)
			return true
		}
		if removeNodeLocked(&n.Children, id) {
			return true
		}
	}
	return false
}

// FindNode walks the document for id (unlocked helper for callers with doc).
func FindNode(doc *Document, id string) *Node {
	if doc == nil {
		return nil
	}
	return findNodeLocked(doc.Roots, id)
}
