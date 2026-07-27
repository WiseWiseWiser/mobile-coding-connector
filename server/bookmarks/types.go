package bookmarks

// Document is the versioned bookmarks tree stored in bookmarks.json.
type Document struct {
	Version int     `json:"version"`
	Roots   []*Node `json:"roots"`
}

// Node is a folder or url entry in the tree.
type Node struct {
	Type     string  `json:"type"`
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	URL      string  `json:"url,omitempty"`
	Browser  *string `json:"browser"`
	Children []*Node `json:"children,omitempty"`
}

// UpdateOpts patches fields on an existing node.
// ClearBrowser sets browser to nil (inherit global default).
// Browser with empty string also clears when provided without ClearBrowser
// via the HTTP layer; Manager.Update uses ClearBrowser and/or Browser pointer.
type UpdateOpts struct {
	Name         *string
	URL          *string
	Browser      *string
	ClearBrowser bool
}
