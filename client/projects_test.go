package client

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListProjects_dirtyQueryParam(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
	}))
	defer srv.Close()

	c := New(srv.URL, "token")
	if _, err := c.ListProjects(ProjectListOptions{DirtyOnly: true}); err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if gotPath != "/api/projects?all=true&dirty=true" {
		t.Fatalf("request path = %q, want /api/projects?all=true&dirty=true", gotPath)
	}
}

func TestListProjects_allQueryParam(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
	}))
	defer srv.Close()

	c := New(srv.URL, "token")
	if _, err := c.ListProjects(ProjectListOptions{}); err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if gotPath != "/api/projects?all=true" {
		t.Fatalf("request path = %q, want /api/projects?all=true", gotPath)
	}
}