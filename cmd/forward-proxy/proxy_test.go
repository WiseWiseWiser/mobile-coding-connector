package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestForwardProxyHTTP(t *testing.T) {
	var receivedURL string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedURL = r.URL.String()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("upstream response"))
	}))
	defer upstream.Close()

	upstreamURL, _ := url.Parse(upstream.URL)
	fwd := NewForwardProxy(upstreamURL)
	fwdServer := httptest.NewServer(http.HandlerFunc(fwd.ServeHTTP))
	defer fwdServer.Close()

	proxyURL, _ := url.Parse(fwdServer.URL)
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}

	resp, err := client.Get("http://example.com/test")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "upstream response" {
		t.Errorf("unexpected body: %s", body)
	}
	if receivedURL != "http://example.com/test" {
		t.Errorf("upstream received wrong URL: %s", receivedURL)
	}
}

func TestForwardProxyHTTPRemovesHopHeaders(t *testing.T) {
	var receivedHeaders http.Header
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	upstreamURL, _ := url.Parse(upstream.URL)
	fwd := NewForwardProxy(upstreamURL)
	fwdServer := httptest.NewServer(http.HandlerFunc(fwd.ServeHTTP))
	defer fwdServer.Close()

	proxyURL, _ := url.Parse(fwdServer.URL)

	conn, err := net.Dial("tcp", proxyURL.Host)
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	defer conn.Close()

	req := "GET http://example.com/path HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"Proxy-Connection: keep-alive\r\n" +
		"Proxy-Authorization: Basic dGVzdDp0ZXN0\r\n" +
		"Connection: keep-alive\r\n" +
		"\r\n"
	conn.Write([]byte(req))

	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	defer resp.Body.Close()

	if receivedHeaders.Get("Proxy-Connection") != "" {
		t.Error("Proxy-Connection header should have been removed")
	}
	if receivedHeaders.Get("Proxy-Authorization") != "" {
		t.Error("Proxy-Authorization header should have been removed")
	}
}

func TestForwardProxyCONNECT(t *testing.T) {
	squidLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("squid listen: %v", err)
	}
	defer squidLn.Close()

	echoReady := make(chan struct{})
	go func() {
		conn, err := squidLn.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		br := bufio.NewReader(conn)
		line, err := br.ReadString('\n')
		if err != nil {
			return
		}
		if !strings.HasPrefix(line, "CONNECT ") {
			fmt.Fprintf(conn, "HTTP/1.1 400 Bad Request\r\n\r\n")
			return
		}

		for {
			line, err := br.ReadString('\n')
			if err != nil {
				return
			}
			if line == "\r\n" || line == "\n" {
				break
			}
		}

		conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
		close(echoReady)

		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			return
		}
		conn.Write(buf[:n])
	}()

	squidURL := &url.URL{Scheme: "http", Host: squidLn.Addr().String()}
	fwd := NewForwardProxy(squidURL)
	fwdServer := httptest.NewServer(http.HandlerFunc(fwd.ServeHTTP))
	defer fwdServer.Close()

	conn, err := net.Dial("tcp", fwdServer.Listener.Addr().String())
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	defer conn.Close()

	fmt.Fprintf(conn, "CONNECT example.com:443 HTTP/1.1\r\nHost: example.com:443\r\n\r\n")

	br := bufio.NewReader(conn)
	statusLine, err := br.ReadString('\n')
	if err != nil {
		t.Fatalf("read status: %v", err)
	}
	if !strings.Contains(statusLine, "200") {
		t.Fatalf("unexpected status: %s", statusLine)
	}

	for {
		line, err := br.ReadString('\n')
		if err != nil {
			t.Fatalf("read headers: %v", err)
		}
		if line == "\r\n" || line == "\n" {
			break
		}
	}

	<-echoReady

	testData := []byte("hello from client via tunnel")
	conn.Write(testData)

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("read echo: %v", err)
	}
	if string(buf[:n]) != string(testData) {
		t.Errorf("unexpected echo: got %q, want %q", buf[:n], testData)
	}
}

func TestForwardProxyCONNECTNon200(t *testing.T) {
	squidLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("squid listen: %v", err)
	}
	defer squidLn.Close()

	go func() {
		conn, err := squidLn.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		br := bufio.NewReader(conn)
		for {
			line, err := br.ReadString('\n')
			if err != nil {
				return
			}
			if line == "\r\n" || line == "\n" {
				break
			}
		}
		conn.Write([]byte("HTTP/1.1 403 Forbidden\r\nContent-Type: text/plain\r\n\r\naccess denied"))
	}()

	squidURL := &url.URL{Scheme: "http", Host: squidLn.Addr().String()}
	fwd := NewForwardProxy(squidURL)
	fwdServer := httptest.NewServer(http.HandlerFunc(fwd.ServeHTTP))
	defer fwdServer.Close()

	conn, err := net.Dial("tcp", fwdServer.Listener.Addr().String())
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	defer conn.Close()

	fmt.Fprintf(conn, "CONNECT blocked.example.com:443 HTTP/1.1\r\nHost: blocked.example.com:443\r\n\r\n")

	br := bufio.NewReader(conn)
	statusLine, err := br.ReadString('\n')
	if err != nil {
		t.Fatalf("read status: %v", err)
	}
	if !strings.Contains(statusLine, "403") {
		t.Errorf("expected 403, got: %s", statusLine)
	}
}

func TestForwardProxyCONNECTDefaultPort(t *testing.T) {
	squidLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("squid listen: %v", err)
	}
	defer squidLn.Close()

	var connectTarget string
	go func() {
		conn, err := squidLn.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		br := bufio.NewReader(conn)
		line, err := br.ReadString('\n')
		if err != nil {
			return
		}
		fmt.Sscanf(line, "CONNECT %s", &connectTarget)
		connectTarget = strings.TrimSpace(connectTarget)

		for {
			line, err := br.ReadString('\n')
			if err != nil {
				return
			}
			if line == "\r\n" || line == "\n" {
				break
			}
		}
		conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	}()

	squidURL := &url.URL{Scheme: "http", Host: squidLn.Addr().String()}
	fwd := NewForwardProxy(squidURL)
	fwdServer := httptest.NewServer(http.HandlerFunc(fwd.ServeHTTP))
	defer fwdServer.Close()

	conn, err := net.Dial("tcp", fwdServer.Listener.Addr().String())
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	defer conn.Close()

	fmt.Fprintf(conn, "CONNECT example.com HTTP/1.1\r\nHost: example.com\r\n\r\n")

	br := bufio.NewReader(conn)
	statusLine, err := br.ReadString('\n')
	if err != nil {
		t.Fatalf("read status: %v", err)
	}
	if !strings.Contains(statusLine, "200") {
		t.Fatalf("unexpected status: %s", statusLine)
	}

	for {
		line, err := br.ReadString('\n')
		if err != nil {
			t.Fatalf("read headers: %v", err)
		}
		if line == "\r\n" || line == "\n" {
			break
		}
	}

	if !strings.Contains(connectTarget, ":443") {
		t.Errorf("expected target with port 443, got: %s", connectTarget)
	}
}
