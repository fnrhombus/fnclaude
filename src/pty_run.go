package main

import (
	"io"
	"os"
	"regexp"
)

// ringBuffer is a fixed-capacity circular byte buffer.  Writes that overflow
// the capacity discard the oldest data.  Only the most recent cap bytes are
// kept, which is all we need for post-exit pattern scanning.
type ringBuffer struct {
	buf  []byte
	cap  int
	pos  int
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

// crossCwdRe matches the cd-and-resume line claude prints when the selected
// session belongs to a different directory. Compiled once at package init.
//
// We can't anchor on the "This conversation is from a different directory."
// preamble: claude's TUI emits cursor-right escapes (e.g. `\x1b[1C`) between
// words instead of literal spaces, so that sentence is never plain-text in
// the PTY stream. The "To resume, run:" line, by contrast, is rendered as
// plain ASCII with real spaces, as is the `cd <path> && claude --resume <uuid>`
// command — both anchors survive the TUI rendering intact.
//
// The `[\s\S]*?` between anchors swallows whatever ANSI / CR / cursor-move
// goo appears between the two lines (varies by terminal width and TUI
// layout — observed: `\x1b[K\r\x1b[1C\x1b[1B`).
var crossCwdRe = regexp.MustCompile(
	`To resume, run:[\s\S]*?cd (\S+) && claude --resume ([0-9a-fA-F-]{36})`,
)

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
