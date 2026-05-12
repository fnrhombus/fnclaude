//go:build !windows

package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"
)

// runWithPTY runs claudeArgv in launchCWD under a PTY.  It:
//   - allocates a PTY and starts the child under it,
//   - forwards stdin → PTY master,
//   - tees PTY output → os.Stdout and a 4 KB ring buffer,
//   - forwards SIGWINCH → PTY master (terminal resize),
//   - returns the child's exit code and the ring buffer contents after exit.
//
// The ring buffer is used by the caller to check for the cross-cwd pattern.
func runWithPTY(claudeArgv []string, launchCWD string) (exitCode int, tail []byte) {
	claudeBin, err := exec.LookPath("claude")
	if err != nil {
		fmt.Fprintf(os.Stderr, "fnclaude: claude not found in PATH: %v\n", err)
		return 1, nil
	}

	cmd := exec.Command(claudeBin, claudeArgv[1:]...)
	cmd.Dir = launchCWD
	cmd.Env = os.Environ()

	// Start the command under a PTY.
	ptmx, err := pty.Start(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fnclaude: failed to start claude with PTY: %v\n", err)
		return 1, nil
	}
	defer func() { _ = ptmx.Close() }()

	// Set the PTY to match the current terminal size.
	if sz, err := pty.GetsizeFull(os.Stdin); err == nil {
		_ = pty.Setsize(ptmx, sz)
	}

	// Put the controlling terminal into raw mode so the PTY behaves
	// transparently (key-by-key, no local echo, etc.).
	if term.IsTerminal(int(os.Stdin.Fd())) {
		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err == nil {
			defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }()
		}
	}

	// Forward SIGWINCH (terminal resize) to the PTY.
	winchCh := make(chan os.Signal, 1)
	signal.Notify(winchCh, syscall.SIGWINCH)
	defer signal.Stop(winchCh)
	go func() {
		for range winchCh {
			if sz, err := pty.GetsizeFull(os.Stdin); err == nil {
				_ = pty.Setsize(ptmx, sz)
			}
		}
	}()

	// Pump stdin → PTY master.
	go func() {
		_, _ = io.Copy(ptmx, os.Stdin)
	}()

	// Ring buffer: keep the last 4 KB of output for pattern scanning.
	ring := newRingBuffer(4 * 1024)

	// Tee PTY output → stdout + ring buffer.
	_, _ = io.Copy(io.MultiWriter(os.Stdout, ring), ptmx)

	// Wait for the child to exit and collect the exit code.
	exitCode = 0
	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				exitCode = status.ExitStatus()
			} else {
				exitCode = 1
			}
		}
		// io.EOF on the PTY master is normal after child exit — not an error.
	}

	return exitCode, ring.Bytes()
}

// silentRelaunch replaces the current process image with a fresh fnclaude
// invocation, using dest as the new cwd and uuid as the session to resume.
// It never returns on success; on failure it writes to stderr and returns.
func silentRelaunch(origArgs []string, dest, uuid string) {
	self, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fnclaude: os.Executable failed, cannot relaunch: %v\n", err)
		return
	}

	newArgs := reconstructArgv(origArgs, dest, uuid)
	// syscall.Exec argv[0] is conventionally the program name.
	argv := append([]string{self}, newArgs...)

	clearScreen()

	if err := syscall.Exec(self, argv, os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "fnclaude: exec relaunch failed: %v\n", err)
	}
}
