package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "serve" {
		os.Exit(1)
	}
	port := 0
	for i := 2; i < len(os.Args)-1; i++ {
		if os.Args[i] == "--port" {
			fmt.Sscanf(os.Args[i+1], "%d", &port)
			break
		}
	}
	if port <= 0 {
		os.Exit(1)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/global/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			w.Write([]byte(`{"model":""}`))
		case http.MethodPatch:
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, "method", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/config/providers", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"providers":[{"id":"xai","models":{"grok-3":{}}}]}`))
	})
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	srv := &http.Server{Addr: addr, Handler: mux}
	go srv.ListenAndServe()
	for i := 0; i < 100; i++ {
		resp, err := http.Get("http://" + addr + "/global/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				select {}
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	os.Exit(1)
}