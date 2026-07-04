// Port occupier for keep-alive-kill-existing doctests.
// Listens on --port; when --ping is set, serves GET /ping -> pong.
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
)

func main() {
	port := flag.Int("port", 0, "TCP port to bind")
	withPing := flag.Bool("ping", false, "respond to GET /ping with pong")
	flag.Parse()
	if *port <= 0 {
		fmt.Fprintln(os.Stderr, "port required")
		os.Exit(2)
	}
	mux := http.NewServeMux()
	if *withPing {
		mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("pong"))
		})
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}