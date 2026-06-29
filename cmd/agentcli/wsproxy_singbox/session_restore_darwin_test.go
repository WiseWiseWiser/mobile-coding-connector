//go:build darwin

package wsproxy_singbox

import (
	"encoding/json"
	"os"
	"testing"
)

func TestTunSessionSnapshotRoundTrip(t *testing.T) {
	if err := saveTunSessionSnapshot("Wi-Fi", []string{"1.1.1.1"}, true, serviceProxyState{}, false); err != nil {
		t.Fatal(err)
	}
	path, err := tunSessionStatePath()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var loaded tunSessionSnapshot
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatal(err)
	}
	if loaded.NetworkService != "Wi-Fi" || !loaded.DNSTouched {
		t.Fatalf("loaded = %#v", loaded)
	}
	clearTunSessionSnapshot()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("snapshot should be cleared, err=%v", err)
	}
}