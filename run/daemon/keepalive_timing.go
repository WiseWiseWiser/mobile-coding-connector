package daemon

import "time"

var keepaliveProcessStart = time.Now()

func keepaliveElapsedMs() int {
	return int(time.Since(keepaliveProcessStart).Milliseconds())
}