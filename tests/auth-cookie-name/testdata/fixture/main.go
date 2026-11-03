// A test-only process boundary for two independent management cookie configurations.
// It uses production auth and origin-gate handlers, never the full server's
// background services, public providers, user HOME, or existing listeners.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/xhd2015/ai-critic/server/auth"
	"github.com/xhd2015/ai-critic/server/proxy/origin"
	"github.com/xhd2015/ai-critic/server/services/authproxy"
)

func main() {
	name := flag.String("auth-cookie-name", auth.DefaultCookieName, "instance cookie name")
	credentials := flag.String("credentials-file", "", "fixture credentials")
	backend := flag.String("backend", "", "optional upstream fixture")
	flag.Parse()
	if err := auth.SetCookieName(*name); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	auth.SetCredentialsFile(*credentials)
	mux := http.NewServeMux()
	auth.RegisterAPI(mux)
	mux.HandleFunc("/api/private", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/_fixture/headers", func(w http.ResponseWriter, r *http.Request) {
		names := []string{}
		for _, cookie := range r.Cookies() {
			names = append(names, cookie.Name)
		}
		json.NewEncoder(w).Encode(map[string]any{"cookies": names, "bearer": r.Header.Get("Authorization") != ""})
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	srv := &http.Server{Handler: auth.Middleware(mux, []string{"/api/login", "/api/auth/check", "/api/auth/status"}), ReadHeaderTimeout: 5 * time.Second}
	go srv.Serve(ln)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()
	urls := map[string]string{"management": "http://" + ln.Addr().String()}
	if *backend != "" {
		proxy, err := authproxy.Start(authproxy.Config{DisableAuth: true, BackendURL: *backend, Transport: origin.NewTransport()})
		if err != nil {
			panic(err)
		}
		defer proxy.Close()
		urls["proxy"] = "http://127.0.0.1:" + strconv.Itoa(proxy.Port())
	}
	json.NewEncoder(os.Stdout).Encode(urls)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
}
