package portforward

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/server/config"
	"github.com/xhd2015/ai-critic/server/domains"
)

func TestPortEventsRetainAndRefreshBootstrapDomains(t *testing.T) {
	original := defaultManager
	defaultManager = NewManager()
	t.Cleanup(func() { defaultManager = original })
	oldPort := domains.GetServerPort()
	domains.SetServerPort(23712)
	domains.SetDomainsFile(filepath.Join(t.TempDir(), "domains.json"))
	t.Cleanup(func() { domains.SetServerPort(oldPort); domains.SetDomainsFile(config.DomainsFile) })
	cfg := &domains.DomainsConfig{Domains: []domains.DomainEntry{{Domain: "first.example", Provider: domains.ProviderCloudflare}}}
	if err := domains.SaveDomains(cfg); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(handlePortEvents))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	scan := bufio.NewScanner(response.Body)
	readEvent := func() []PortForward {
		t.Helper()
		for scan.Scan() {
			line := scan.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			var rows []PortForward
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &rows); err != nil {
				t.Fatal(err)
			}
			return rows
		}
		t.Fatalf("event stream ended: %v", scan.Err())
		return nil
	}
	has := func(rows []PortForward, label string) bool {
		for _, p := range rows {
			if p.Label == label {
				return true
			}
		}
		return false
	}
	if rows := readEvent(); !has(rows, "first.example") {
		t.Fatalf("initial: %+v", rows)
	}
	cfg.Domains = append(cfg.Domains, domains.DomainEntry{Domain: "second.example", Provider: domains.ProviderCloudflare})
	if err := domains.SaveDomains(cfg); err != nil {
		t.Fatal(err)
	}
	if rows := readEvent(); !has(rows, "first.example") || !has(rows, "second.example") {
		t.Fatalf("bootstrap poll: %+v", rows)
	}
	defaultManager.mu.Lock()
	defaultManager.tunnels[8080] = &tunnel{port: 8080, label: "local", status: StatusActive}
	defaultManager.notifySubscribers()
	defaultManager.mu.Unlock()
	if rows := readEvent(); !has(rows, "first.example") || !has(rows, "second.example") || !has(rows, "local") {
		t.Fatalf("notification dropped bootstrap domains: %+v", rows)
	}
}
