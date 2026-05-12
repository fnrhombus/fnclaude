package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"
)

// ringBuffer is a fixed-capacity circular byte buffer.  Writes that overflow
// the capacity discard the oldest data.  Only the most recent cap bytes are
// kept, which is all we need for post-exit pattern scanning.
type ringBuffer struct {
	buf []byte
	cap int
	pos int
	full bool
}

func newRingBuffer(capacity int) *ringBuffer {
	return &ringBuffer{buf: make([]byte, capacity), cap: capacity}
}

// Write appends p to the ring, dropping oldest bytes when full.
func (r *ringBuffer) Write(p []byte) (int, error) {
	for _, b := range p {
		r.buf[r.pos] = b
		r.pos = (r.pos + 1) % r.cap
		if r.pos == 0 {
			r.full = true
		}
	}
	return len(p), nil
}

// Bytes returns the ring contents in order (oldest first).
func (r *ringBuffer) Bytes() []byte {
	if !r.full {
		return r.buf[:r.pos]
	}
	out := make([]byte, r.cap)
	copy(out, r.buf[r.pos:])
	copy(out[r.cap-r.pos:], r.buf[:r.pos])
	return out
}

// crossCwdRe matches the message claude prints when the selected session
// belongs to a different directory.  Compiled once at package init.
var crossCwdRe = regexp.MustCompile(
	`This conversation is from a different directory\.\s+To resume, run:\s+cd (\S+) && claude --resume ([0-9a-fA-F-]{36})`,
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

// detectCrossCwd searches tail (the ring buffer contents) for the cross-cwd
// redirect message that claude prints when the user resumes an alien-cwd
// session.  If found, it returns (destDir, uuid, true); otherwise ("", "", false).
//
// When multiple matches appear (unlikely but defensive), the LAST match is
// returned.
func detectCrossCwd(tail []byte) (destDir, uuid string, ok bool) {
	matches := crossCwdRe.FindAllSubmatch(tail, -1)
	if len(matches) == 0 {
		return "", "", false
	}
	last := matches[len(matches)-1]
	return string(last[1]), string(last[2]), true
}

// clearScreen writes the ANSI escape sequence that clears the screen and
// moves the cursor to the top-left.  Called before relaunching to hide the
// brief flicker of the "different directory" message that already scrolled
// to the terminal before we detected it.
func clearScreen() {
	_, _ = os.Stdout.Write([]byte("\033[2J\033[H"))
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

// reconstructArgv builds the new fnclaude argument list when silently
// relaunching after a cross-cwd session resume.
//
// origArgs is os.Args[1:] from the original invocation.  dest is the
// destination directory extracted from claude's message; uuid is the session
// id to resume.
//
// Algorithm:
//  1. Collect leading magic words (model aliases + effort levels) as
//     preserved_magic — these are kept verbatim at the front.
//  2. Skip contiguous non-flag, non-magic tokens (the original path args).
//  3. Collect everything from the first flag-shaped ("-" prefix) token onward
//     as rest — preserved verbatim at the end.
//
// Result: preserved_magic + [dest] + ["--resume", uuid] + rest
//
// Note: if the original argv already contained --resume / -r / --continue /
// -c, the picker wouldn't have been shown, the cross-cwd pattern wouldn't
// have been emitted, and this function wouldn't be called.  No special-case
// is needed for those flags.
func reconstructArgv(origArgs []string, dest, uuid string) []string {
	var preservedMagic []string
	var rest []string

	i := 0
	// Phase 1: collect leading magic words.
	for i < len(origArgs) {
		tok := origArgs[i]
		if isMagicWord(tok) {
			preservedMagic = append(preservedMagic, tok)
			i++
			continue
		}
		break
	}

	// Phase 2: skip positional path tokens (non-flag, non-magic).
	for i < len(origArgs) {
		tok := origArgs[i]
		if isFlag(tok) {
			break // reached flags — stop skipping paths
		}
		// Non-flag, non-magic: this was a path token; skip it.
		i++
	}

	// Phase 3: everything else is rest.
	rest = origArgs[i:]

	// Assemble: magic + dest + --resume uuid + rest
	out := make([]string, 0, len(preservedMagic)+3+len(rest))
	out = append(out, preservedMagic...)
	out = append(out, dest)
	out = append(out, "--resume", uuid)
	out = append(out, rest...)
	return out
}

// isMagicWord reports whether tok is a model alias or effort level.
func isMagicWord(tok string) bool {
	return modelAliases[tok] || effortLevels[tok]
}

// isFlag reports whether tok is flag-shaped (starts with "-").
func isFlag(tok string) bool {
	return len(tok) > 0 && tok[0] == '-'
}

// assertRingWriter is a compile-time check that ringBuffer implements io.Writer.
var _ io.Writer = (*ringBuffer)(nil)
