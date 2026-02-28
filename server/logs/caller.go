package logs

import (
	"fmt"
	"runtime"
)

func PrintCallerStack() {
	pc := make([]uintptr, 10)
	n := runtime.Callers(2, pc)
	frames := runtime.CallersFrames(pc[:n])
	for {
		frame, more := frames.Next()
		fmt.Printf("  %s:%d %s\n", frame.File, frame.Line, frame.Function)
		if !more {
			break
		}
	}
}
