package main

import (
	"fmt"
	"os"
)

// deferredWarnings accumulates non-fatal warnings issued during fnclaude
// setup so they can be flushed AFTER claude exits. Warnings printed before
// claude launches scroll off-screen too fast to read; flushing on exit shows
// them in the user's shell where they have time to actually be seen.
//
// Fatal errors that prevent launch entirely (e.g. claude binary not on PATH)
// should still print directly to stderr and exit 1 — they don't need
// deferring because there's no claude session to drown them out.
var deferredWarnings []string

// warn queues a non-fatal warning. Format string and args mirror fmt.Sprintf.
func warn(format string, args ...any) {
	deferredWarnings = append(deferredWarnings, fmt.Sprintf(format, args...))
}

// flushWarnings prints all queued warnings to stderr in order, prefixed for
// clarity. Called from run() after claude exits.
func flushWarnings() {
	for _, w := range deferredWarnings {
		fmt.Fprintln(os.Stderr, w)
	}
}
