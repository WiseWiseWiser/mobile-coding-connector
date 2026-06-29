package server

import (
	"fmt"
	"time"
)

var processStart = time.Now()

func bootstrapElapsedMs() int {
	return int(time.Since(processStart).Milliseconds())
}

func logBootstrapPhase(phase string, port int, extra string) {
	if port > 0 {
		if extra != "" {
			fmt.Printf("[bootstrap] phase=%s t_ms=%d port=%d %s\n", phase, bootstrapElapsedMs(), port, extra)
			return
		}
		fmt.Printf("[bootstrap] phase=%s t_ms=%d port=%d\n", phase, bootstrapElapsedMs(), port)
		return
	}
	if extra != "" {
		fmt.Printf("[bootstrap] phase=%s t_ms=%d %s\n", phase, bootstrapElapsedMs(), extra)
		return
	}
	fmt.Printf("[bootstrap] phase=%s t_ms=%d\n", phase, bootstrapElapsedMs())
}