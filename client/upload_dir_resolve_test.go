package client

import "testing"

func TestResolveEffectiveUploadDir_Missing(t *testing.T) {
	got, err := ResolveEffectiveUploadDir("/tmp/myapp", "/root/apps/myapp", &PathInfo{Exists: false})
	if err != nil {
		t.Fatal(err)
	}
	if got != "/root/apps/myapp" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveEffectiveUploadDir_ExistingDirNestsBasename(t *testing.T) {
	got, err := ResolveEffectiveUploadDir("/tmp/myapp", "/root/apps", &PathInfo{Exists: true, IsDir: true})
	if err != nil {
		t.Fatal(err)
	}
	if got != "/root/apps/myapp" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveEffectiveUploadDir_ExistingFileErrors(t *testing.T) {
	_, err := ResolveEffectiveUploadDir("/tmp/myapp", "/root/apps/myapp", &PathInfo{Exists: true, IsDir: false})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveUploadDirTarget_TrailingSlash(t *testing.T) {
	got := ResolveUploadDirTarget("/tmp/myapp", "uploads/", "/root")
	if got != "/root/uploads" {
		t.Fatalf("got %q", got)
	}
}
