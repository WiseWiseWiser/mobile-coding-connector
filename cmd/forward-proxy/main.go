package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/xhd2015/less-gen/flags"
)

const help = `Usage: forward-proxy [options]

A forward HTTP/HTTPS proxy that chains requests through an upstream proxy.

Options:
  --listen ADDR          Address to listen on (default: :8888)
  --upstream-proxy URL   Upstream proxy URL (required, e.g. http://proxy:3128)
  -h, --help             Show this help message
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	var listenAddr string = ":8888"
	var upstreamProxy string

	args, err := flags.
		String("--listen", &listenAddr).
		String("--upstream-proxy", &upstreamProxy).
		Help("-h,--help", help).
		Parse(args)
	if err != nil {
		return err
	}

	if upstreamProxy == "" {
		return fmt.Errorf("--upstream-proxy is required")
	}

	upstreamURL, err := url.Parse(upstreamProxy)
	if err != nil {
		return fmt.Errorf("invalid upstream proxy URL: %w", err)
	}

	proxy := NewForwardProxy(upstreamURL)

	server := &http.Server{
		Addr:    listenAddr,
		Handler: http.HandlerFunc(proxy.ServeHTTP),
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		server.Close()
	}()

	fmt.Printf("Forward proxy listening on %s, chaining through %s\n", listenAddr, upstreamProxy)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}
