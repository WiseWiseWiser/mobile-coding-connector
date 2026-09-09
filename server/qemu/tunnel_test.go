package qemu

import "testing"

func TestMarkDomainActive(t *testing.T) {
	MarkDomainInactive("example.test")
	if IsDomainActive("example.test") {
		t.Fatal("expected inactive")
	}
	MarkDomainActive("example.test", "core")
	if !IsDomainActive("example.test") {
		t.Fatal("expected active")
	}
	MarkDomainInactive("example.test")
	if IsDomainActive("example.test") {
		t.Fatal("expected inactive after clear")
	}
}
