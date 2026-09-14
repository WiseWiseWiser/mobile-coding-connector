package client

import (
	"net/url"
	"testing"
)

func TestPathUnescapeWorkspaceKey(t *testing.T) {
	key := url.PathEscape("/root/seatalk-local-bot")
	got, err := url.PathUnescape(key)
	if err != nil {
		t.Fatal(err)
	}
	if got != "/root/seatalk-local-bot" {
		t.Fatalf("got %q", got)
	}
}
