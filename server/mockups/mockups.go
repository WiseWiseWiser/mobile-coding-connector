// Package mockups serves UI design mockups (plain HTML) that are embedded in
// the binary. They exist so a design can be reviewed on a phone through the
// same tunnel as the app before any React code is written.
package mockups

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed html/*.html
var htmlFS embed.FS

// RegisterAPI exposes the embedded mockups under /mockups/.
func RegisterAPI(mux *http.ServeMux) {
	sub, err := fs.Sub(htmlFS, "html")
	if err != nil {
		// Cannot happen: the directory is guaranteed by the go:embed pattern.
		panic(err)
	}
	mux.Handle("/mockups/", http.StripPrefix("/mockups/", http.FileServer(http.FS(sub))))
}
