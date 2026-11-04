package portforward

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestTunnelConnectPipesRawBytes verifies the CONNECT hop: the client's raw
// stream reaches the origin target untouched after the 200 line.
func TestTunnelConnectPipesRawBytes(t *testing.T) {
	// Origin "target": raw TCP echo.
	echoLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer echoLn.Close()
	go func() {
		for {
			conn, err := echoLn.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				_, _ = io.Copy(c, c)
				_ = c.Close()
			}(conn)
		}
	}()

	hop := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodConnect {
			http.Error(w, "expect CONNECT", http.StatusBadRequest)
			return
		}
		tunnelConnect(w, r, echoLn.Addr().String())
	}))
	defer hop.Close()

	// Raw client: send CONNECT to the hop, then use the tunnel.
	conn, err := net.DialTimeout("tcp", strings.TrimPrefix(hop.URL, "http://"), 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	fmt.Fprintf(conn, "CONNECT anything:443 HTTP/1.1\r\nHost: anything:443\r\n\r\n")
	br := bufio.NewReader(conn)
	statusLine, err := br.ReadString('\n')
	if err != nil {
		t.Fatalf("read status: %v", err)
	}
	if !strings.Contains(statusLine, "200") {
		t.Fatalf("status = %q, want 200", statusLine)
	}
	fmt.Fprint(conn, "tunnel-ping")
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	reply, err := io.ReadAll(conn)
	if err != nil && len(reply) == 0 {
		t.Fatalf("read echo: %v", err)
	}
	if string(reply) != "tunnel-ping" {
		t.Fatalf("echo = %q", string(reply))
	}
}
