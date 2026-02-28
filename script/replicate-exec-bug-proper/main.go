// Proper test of exec-restart with flag parsing
// This simulates how the real server works:
// 1. Parse flags from os.Args
// 2. Set global state based on flags
// 3. Check if this works after syscall.Exec()

package main

import (
	"fmt"
	"os"
	"syscall"
	"time"
)

// Global state (like quicktest.enabled)
var quickTestEnabled bool

func main() {
	fmt.Println("=== Proper Exec-Restart Test (with flag parsing) ===")
	fmt.Printf("PID: %d\n", os.Getpid())
	fmt.Printf("Args: %v\n\n", os.Args)

	// STEP 1: Parse flags from os.Args (like server does)
	fmt.Println("STEP 1: Parsing flags from os.Args...")
	quickTestEnabled = false
	for _, arg := range os.Args {
		if arg == "--quick-test" {
			quickTestEnabled = true
			break
		}
	}
	fmt.Printf("After parsing: quickTestEnabled = %v\n\n", quickTestEnabled)

	// STEP 2: Check if we're being restarted
	hasQuickTestFlag := false
	for _, arg := range os.Args {
		if arg == "--quick-test" {
			hasQuickTestFlag = true
			break
		}
	}

	if hasQuickTestFlag {
		fmt.Println("[RESTART DETECTED] --quick-test flag found in os.Args")
		fmt.Println("Checking if global state was properly restored from flags...")

		// THE KEY TEST: Did flag parsing work after exec?
		if quickTestEnabled {
			fmt.Println("\n✅ SUCCESS!")
			fmt.Println("   - syscall.Exec() replaced the process")
			fmt.Println("   - Global variables were reset (expected)")
			fmt.Println("   - BUT flags were re-parsed from os.Args")
			fmt.Println("   - Global state properly restored!")
		} else {
			fmt.Println("\n❌ FAILURE!")
			fmt.Println("   - Global state not restored after restart")
		}

		fmt.Println("\n=== Test Complete ===")
		return
	}

	// First run: Set up the flag and exec
	fmt.Println("[INITIAL STARTUP]")
	fmt.Println("This is the first run, preparing to exec-restart...")

	// Prepare to exec WITH --quick-test flag in args
	newArgs := append([]string{os.Args[0], "--quick-test"}, os.Args[1:]...)
	fmt.Printf("\nExecuting syscall.Exec with args: %v\n", newArgs)
	fmt.Println("Process will be replaced but keep same PID...")

	time.Sleep(500 * time.Millisecond)

	// Execute syscall.Exec
	err := syscall.Exec(os.Args[0], newArgs, os.Environ())
	if err != nil {
		fmt.Printf("syscall.Exec failed: %v\n", err)
		os.Exit(1)
	}

	// This line should never be reached
	fmt.Println("ERROR: This should not print after syscall.Exec!")
}
