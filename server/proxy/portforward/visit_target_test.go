package portforward

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/server/config"
)

func TestVisitTargetActiveSSEPreventsIdleExpiry(t *testing.T) {
	done := make(chan struct{})
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: first\n\n"))
		w.(http.Flusher).Flush()
		<-r.Context().Done()
		close(done)
	}))
	defer up.Close()
	u, _ := url.Parse(up.URL)
	port, _ := strconv.Atoi(u.Port())
	m := NewVisitSessionManager()
	m.RegisterProvider(&testProvider{name: ProviderCloudflareOwned})
	now := time.Now()
	m.SetNow(func() time.Time { return now })
	s, err := m.StartTarget(config.OriginTarget{Host: u.Hostname(), Scheme: "http", Port: port}, "owned", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Stop(s.ID)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", fmtHop(s.ProxyPort), nil)
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	line, err := bufio.NewReader(response.Body).ReadString('\n')
	if err != nil || line != "data: first\n" {
		t.Fatalf("visit SSE buffered %q %v", line, err)
	}
	m.SetNow(func() time.Time { return now.Add(time.Hour) })
	m.Sweep()
	if len(m.List()) != 1 {
		t.Fatal("active SSE expired")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("visit did not cancel upstream")
	}
	deadline := time.Now().Add(time.Second)
	for {
		m.mu.Lock()
		active := m.sessions[s.ID].active
		m.mu.Unlock()
		if active == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("visit request stuck active")
		}
		time.Sleep(time.Millisecond)
	}
	m.SetNow(func() time.Time { return now.Add(2 * time.Hour) })
	m.Sweep()
	if len(m.List()) != 0 {
		t.Fatal("idle visit not expired after SSE ended")
	}
}
func TestVisitSamePortDifferentOriginsRequireID(t *testing.T) {
	m := NewVisitSessionManager()
	m.RegisterProvider(&testProvider{name: ProviderCloudflareOwned})
	a, err := m.StartTarget(config.OriginTarget{Host: "192.168.1.1", Port: 3080}, "owned", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Stop(a.ID)
	b, err := m.StartTarget(config.OriginTarget{Host: "192.168.1.2", Port: 3080}, "owned", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Stop(b.ID)
	if len(m.List()) != 2 {
		t.Fatal("visit collision")
	}
	if err := m.Stop("3080"); err == nil {
		t.Fatal("ambiguous numeric visit stop accepted")
	}
}
