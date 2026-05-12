//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
)

// runWithPTY (Windows): no PTY, no ring-buffer scanning. claude is spawned
// with inherited stdio and the return tail is nil so detectCrossCwd never
// matches — cross-cwd-resume is a no-op on Windows for now.
//
// This is intentional: Windows console plumbing (ConPTY etc.) is its own
// project and not on the v1 roadmap. fnclaude still launches and exits
// correctly; only the silent-relaunch feature is unavailable.
func runWithPTY(claudeArgv []string, launchCWD string) (exitCode int, tail []byte) {
	claudeBin, err := exec.LookPath("claude")
	if err != nil {
		fmt.Fprintf(os.Stderr, "fnclaude: claude not found in PATH: %v\n", err)
		return 1, nil
	}

	cmd := exec.Command(claudeBin, claudeArgv[1:]...)
	cmd.Dir = launchCWD
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return 1, nil
	}
	return 0, nil
}

// silentRelaunch (Windows): unreachable in practice — detectCrossCwd never
// matches on Windows because runWithPTY returns a nil tail. Defensive stub:
// surfaces a clear error if something upstream changes and this path fires.
func silentRelaunch(origArgs []string, dest, uuid string) {
	fmt.Fprintln(os.Stderr, "fnclaude: cross-cwd-resume relaunch is not supported on Windows")
}
