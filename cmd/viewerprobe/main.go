package main

// Viewer probe: connects to /api/browser/ws like the phone does, keeps the
// capture loop alive, and prints every status error it sees — used to verify
// a transient navigation failure appears and then clears itself.

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	token := ""
	if b, err := os.ReadFile(os.Getenv("HOME") + "/.ai-critic/server-credentials"); err == nil {
		if len(b) > 0 {
			// first line is the bearer token
			for i, c := range string(b) {
				if c == '\r' || c == '\n' {
					token = string(b[:i])
					break
				}
			}
			if token == "" {
				token = string(b)
			}
		}
	}
	header := http.Header{"Authorization": {"Bearer " + token}}
	url := "ws://127.0.0.1:23712/api/browser/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, header)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer conn.Close()
	fmt.Println("connected as viewer; hub capture loop should start")
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("read end:", err)
			return
		}
		if len(data) > 0 && data[0] == 0xFF || len(data) > 2 && data[0]=='\x00' {
			continue // binary frame
		}
		s := string(data)
		if len(s) > 260 {
			s = s[:260]
		}
		fmt.Println(time.Now().Format("15:04:05"), s)
	}
}
