package logs

import (
	"fmt"
	"runtime"
)

func PrintCallerStack() {
	printCallerStack(2)
}

// PrintCallerStackSkip prints the caller stack, skipping the given number of frames
// on top of the standard 2 (runtime.Callers + this function).
func PrintCallerStackSkip(extraSkip int) {
	printCallerStack(2 + extraSkip)
}

func printCallerStack(skip int) {
	pc := make([]uintptr, 10)
	n := runtime.Callers(skip, pc)
	frames := runtime.CallersFrames(pc[:n])
	for {
		frame, more := frames.Next()
		fmt.Printf("  %s:%d %s\n", frame.File, frame.Line, frame.Function)
		if !more {
			break
		}
	}
}
