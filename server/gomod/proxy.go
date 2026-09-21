package gomod

import (
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// NewProxyHandler serves the GOPROXY protocol over an on-disk GOMODCACHE
// `cache/download` tree. Layout maps 1:1 to the protocol:
//
//	GET $base/$escaped-module/@v/list      -> generated from the @v dir
//	GET $base/$escaped-module/@latest      -> highest version's .info (synthesized if missing)
//	GET $base/$escaped-module/@v/vX.info   -> file, or synthesized {"Version":"vX"} when .mod/.zip exists
//	GET $base/$escaped-module/@v/vX.mod    -> file
//	GET $base/$escaped-module/@v/vX.zip    -> file
//
// Anything missing returns 404 so a GOPROXY chain falls through to the next entry.
func NewProxyHandler(root string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := strings.Trim(path.Clean("/"+r.URL.Path), "/")
		if rel == "" || rel == "." || strings.HasPrefix(rel, "..") {
			http.NotFound(w, r)
			return
		}
		switch {
		case strings.HasSuffix(rel, "/@v/list"):
			serveList(w, r, filepath.Join(root, filepath.FromSlash(strings.TrimSuffix(rel, "list"))))
		case strings.HasSuffix(rel, "/@latest"):
			serveLatest(w, r, filepath.Join(root, filepath.FromSlash(strings.TrimSuffix(rel, "@latest"))))
		case strings.HasSuffix(rel, ".info"):
			full := filepath.Join(root, filepath.FromSlash(rel))
			if _, err := os.Stat(full); err != nil {
				// On-disk caches often lack .info for transitive versions;
				// synthesize one when a .mod or .zip exists for that version.
				if hasModOrZip(filepath.Dir(full), strings.TrimSuffix(path.Base(rel), ".info")) {
					w.Header().Set("Content-Type", "application/json")
					_, _ = io.WriteString(w, `{"Version":"`+strings.TrimSuffix(path.Base(rel), ".info")+`"}`)
					return
				}
			}
			http.ServeFile(w, r, full)
		default:
			http.ServeFile(w, r, filepath.Join(root, filepath.FromSlash(rel)))
		}
	})
}

func serveList(w http.ResponseWriter, r *http.Request, dir string) {
	vers, err := versions(dir)
	if err != nil || len(vers) == 0 {
		// 200 with empty body means "no versions known" (valid GOPROXY response);
		// a missing dir returns 404 below via empty body as well.
		w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	for _, v := range vers {
		_, _ = io.WriteString(w, v+"\n")
	}
}

func serveLatest(w http.ResponseWriter, r *http.Request, modDir string) {
	vers, err := versions(filepath.Join(modDir, "@v"))
	if err != nil || len(vers) == 0 {
		http.NotFound(w, r)
		return
	}
	latest := vers[len(vers)-1]
	data, err := os.ReadFile(filepath.Join(modDir, "@v", latest+".info"))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"Version":"`+latest+`"}`)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}

// versions reads a @v dir and returns sorted unique versions
// (from .info/.mod/.zip file basenames).
func versions(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool, len(entries))
	vers := make([]string, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		for _, ext := range []string{".info", ".mod", ".zip"} {
			if strings.HasSuffix(name, ext) {
				v := strings.TrimSuffix(name, ext)
				if v != "" && !seen[v] {
					seen[v] = true
					vers = append(vers, v)
				}
			}
		}
	}
	sort.Slice(vers, func(i, j int) bool { return semverLess(vers[i], vers[j]) })
	return vers, nil
}

func hasModOrZip(dir, v string) bool {
	for _, ext := range []string{".mod", ".zip"} {
		if _, err := os.Stat(filepath.Join(dir, v+ext)); err == nil {
			return true
		}
	}
	return false
}

func semverLess(a, b string) bool {
	ca, pa := splitPre(a)
	cb, pb := splitPre(b)
	if x := compareCore(ca, cb); x != 0 {
		return x < 0
	}
	// no prerelease > prerelease
	if pa == "" && pb != "" {
		return false
	}
	if pa != "" && pb == "" {
		return true
	}
	return pa < pb
}

func splitPre(v string) (string, string) {
	if i := strings.Index(v, "-"); i >= 0 {
		return v[:i], v[i+1:]
	}
	return v, ""
}

func compareCore(a, b string) int {
	as := strings.Split(strings.TrimPrefix(a, "v"), ".")
	bs := strings.Split(strings.TrimPrefix(b, "v"), ".")
	for i := 0; i < 3; i++ {
		ai, _ := strconv.Atoi(as[i])
		bi, _ := strconv.Atoi(bs[i])
		if ai != bi {
			return ai - bi
		}
	}
	return 0
}

// CountModules returns the number of top-level module path dirs under root.
func CountModules(root string) int {
	entries, err := os.ReadDir(root)
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			n++
		}
	}
	return n
}
