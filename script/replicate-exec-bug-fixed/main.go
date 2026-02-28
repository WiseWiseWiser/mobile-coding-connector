// Replicate the exec-restart bug - FIXED VERSION
// This demonstrates that syscall.Exec resets global state BUT
// the command-line flags are preserved in os.Args and can be re-parsed

package main

import (
	"fmt"
	"os"
	"syscall"
	"time"
)

// Global variable that simulates quicktest.enabled
var quickTestEnabled = false

func main() {
	fmt.Println("=== Exec-Restart State Recovery Test ===")
	fmt.Printf("PID: %d\n", os.Getpid())
	fmt.Printf("All Args: %v\n", os.Args)
	fmt.Printf("Number of args: %d\n", len(os.Args))

	// Check each arg
	for i, arg := range os.Args {
		fmt.Printf("  Arg[%d]: %s\n", i, arg)
	}

	// Check if we're being restarted (look for --quick-test in args)
	hasQuickTestFlag := false
	for _, arg := range os.Args {
		if arg == "--quick-test" {
			hasQuickTestFlag = true
			break
		}
	}

	if hasQuickTestFlag {
		fmt.Println("\n[RESTART DETECTED] --quick-test flag found in os.Args")
		fmt.Println("Checking global state...")

		// THE KEY POINT: Global var is reset, but we can recover from args
		if quickTestEnabled {
			fmt.Println("✅ Global quickTestEnabled = true (state preserved - UNEXPECTED)")
		} else {
			fmt.Println("⚠️  Global quickTestEnabled = false (state reset - EXPECTED)")
			fmt.Println("\n🔧 RECOVERY: Can restore state from os.Args")

			// Demonstrate recovery
			fmt.Println("\nRecovering state from command-line args...")
			for _, arg := range os.Args {
				if arg == "--quick-test" {
					quickTestEnabled = true
					fmt.Println("✅ Restored quickTestEnabled = true from --quick-test flag")
					break
				}
			}
		}

		fmt.Println("\n=== Test Complete ===")
		fmt.Println("\nCONCLUSION:")
		fmt.Println("- syscall.Exec() resets global variables (expected)")
		fmt.Println("- BUT os.Args is preserved with all flags")
		fmt.Println("- Server CAN recover quick-test mode by re-parsing os.Args")
		fmt.Println("\nTHE BUG: Server doesn't re-parse --quick-test from os.Args")
		return
	}

	// First run: Set up the flag and exec
	fmt.Println("\n[INITIAL STARTUP]")
	fmt.Println("Setting --quick-test flag in global variable")
	quickTestEnabled = true
	fmt.Printf("Global quickTestEnabled = %v\n", quickTestEnabled)

	// Prepare to exec WITH --quick-test flag in args
	newArgs := append([]string{os.Args[0], "--quick-test"}, os.Args[1:]...)
	fmt.Printf("\nExecuting syscall.Exec with args: %v\n", newArgs)
	fmt.Println("This will replace the current process, preserving PID...")

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
