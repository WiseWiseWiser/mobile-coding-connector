// Replicate the exec-restart bug
// This demonstrates that syscall.Exec resets global state

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
	fmt.Println("=== Exec-Restart Bug Replication ===")
	fmt.Printf("PID: %d\n", os.Getpid())
	fmt.Printf("Args: %v\n", os.Args)

	// Check if we're being restarted
	if len(os.Args) > 1 && os.Args[1] == "--quick-test" {
		fmt.Println("\n[PASS] Restarted with --quick-test flag")
		fmt.Println("Checking global state...")

		// This is the BUG: the flag is present but the global var was reset!
		if quickTestEnabled {
			fmt.Println("[PASS] Global quickTestEnabled = true (correct)")
		} else {
			fmt.Println("[FAIL] Global quickTestEnabled = false (BUG!)")
			fmt.Println("\nThe syscall.Exec() replaced the process,")
			fmt.Println("but global variables were reset to defaults!")
		}

		fmt.Println("\n=== Test Complete ===")
		return
	}

	// First run: Set up the flag and exec
	fmt.Println("\nFirst run: Setting --quick-test flag")
	quickTestEnabled = true
	fmt.Printf("Global quickTestEnabled = %v\n", quickTestEnabled)

	// Prepare to exec with --quick-test flag
	newArgs := append([]string{os.Args[0], "--quick-test"}, os.Args[1:]...)
	fmt.Printf("\nExecuting syscall.Exec with args: %v\n", newArgs)
	fmt.Println("This should replace the current process...")

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
