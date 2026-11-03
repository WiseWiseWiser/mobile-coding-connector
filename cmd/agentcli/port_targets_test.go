package agentcli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xhd2015/ai-critic/client"
)

func TestTargetMappingCLIAddsWithoutLocalListeningScan(t *testing.T) {
	var request map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/ports" || r.Method != "POST" {
			t.Errorf("unexpected call %s %s", r.Method, r.URL)
			http.Error(w, "unexpected", 500)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&request)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"map_mac","localPort":3080,"target":{"scheme":"http","host":"192.168.1.1","port":3080},"status":"active","publicUrl":"https://mac.example","auth":{"required":true,"token":"should-be-hidden"},"origin":{"tcpReachable":true}}`)
	}))
	defer server.Close()
	var out bytes.Buffer
	var err error
	withStdout(&out, func() {
		err = runPortMapping(func() (*client.Client, error) { return client.New(server.URL, ""), nil }, []string{"add", "3080", "--target-host", "192.168.1.1", "--hostname", "mac.example", "--require-auth", "--json"})
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "should-be-hidden") {
		t.Fatal("token leaked")
	}
	if !strings.Contains(out.String(), "map_mac") {
		t.Fatal("ID missing")
	}
	target := request["target"].(map[string]any)
	if target["host"] != "192.168.1.1" || target["port"] != float64(3080) {
		t.Fatalf("target %v", target)
	}
}
func TestTargetMappingCLIUpdatePreservesUnspecifiedFields(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "GET" && r.URL.Path == "/api/ports/mappings" {
			fmt.Fprint(w, `{"version":2,"mappings":{"map_mac":{"id":"map_mac","target":{"scheme":"http","host":"192.168.1.1","port":3080},"hostname":"mac.example","desired":true}}}`)
			return
		}
		if r.Method != "PUT" || r.URL.Query().Get("id") != "map_mac" {
			t.Errorf("unexpected %s %s", r.Method, r.URL)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		fmt.Fprint(w, `{"id":"map_mac","status":"active"}`)
	}))
	defer server.Close()
	var out bytes.Buffer
	var err error
	withStdout(&out, func() {
		err = runPortMapping(func() (*client.Client, error) { return client.New(server.URL, ""), nil }, []string{"update", "map_mac", "--target-host", "macbook.local"})
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"auth", "label", "provider"} {
		if _, ok := body[key]; ok {
			t.Fatalf("omitted %s overwritten", key)
		}
	}
	target := body["target"].(map[string]any)
	if target["port"] != float64(3080) || target["host"] != "macbook.local" {
		t.Fatal(target)
	}
}
func TestTargetMappingCLIDeletesByID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" || r.URL.Query().Get("id") != "map_mac" || r.URL.Query().Get("forget") != "true" {
			t.Errorf("wrong delete %s %s", r.Method, r.URL)
		}
		fmt.Fprint(w, `{"status":"removed"}`)
	}))
	defer server.Close()
	var out bytes.Buffer
	var err error
	withStdout(&out, func() {
		err = runPortMapping(func() (*client.Client, error) { return client.New(server.URL, ""), nil }, []string{"delete", "map_mac", "--forget"})
	})
	if err != nil {
		t.Fatal(err)
	}
}
