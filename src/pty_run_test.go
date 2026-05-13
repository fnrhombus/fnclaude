package main

import (
	"bytes"
	"os"
	"testing"
)

// ── crossCwdRe tests ──────────────────────────────────────────────────────

// claudeExactMessage is the exact multi-line message claude prints when the
// user picks a session from a different directory via the Ctrl+A picker.
const claudeExactMessage = `This conversation is from a different directory.

To resume, run:
  cd /home/tom/src/arch-setup@fnrhombus && claude --resume 68aa15ae-af23-4c7a-b59f-5cee07c61790

(Command copied to clipboard)`

func TestCrossCwdRe_ExactMessage(t *testing.T) {
	dest, uuid, ok := detectCrossCwd([]byte(claudeExactMessage))
	if !ok {
		t.Fatal("detectCrossCwd: expected match, got none")
	}
	if dest != "/home/tom/src/arch-setup@fnrhombus" {
		t.Errorf("dest: got %q", dest)
	}
	if uuid != "68aa15ae-af23-4c7a-b59f-5cee07c61790" {
		t.Errorf("uuid: got %q", uuid)
	}
}

func TestCrossCwdRe_NoMatch(t *testing.T) {
	_, _, ok := detectCrossCwd([]byte("normal claude output\nno resume message here"))
	if ok {
		t.Error("detectCrossCwd: expected no match, got one")
	}
}

func TestCrossCwdRe_EmptyInput(t *testing.T) {
	_, _, ok := detectCrossCwd(nil)
	if ok {
		t.Error("detectCrossCwd: expected no match on nil, got one")
	}
}

func TestCrossCwdRe_PartialMessage_NoMatch(t *testing.T) {
	partial := "This conversation is from a different directory."
	_, _, ok := detectCrossCwd([]byte(partial))
	if ok {
		t.Error("detectCrossCwd: expected no match for partial message, got one")
	}
}

// TestCrossCwdRe_RealCapture exercises the regex against an actual captured
// PTY stream from a real `claude --resume` → Ctrl-A → cross-cwd-pick session,
// recorded via `script(1)`. The fixture lives at testdata/cross-cwd-tty-capture.bin
// and includes the script(1) header line followed by the raw bytes claude
// emitted, ANSI escapes and all.
//
// This is the strongest regression test we have — if claude ever changes its
// TUI rendering of the cross-cwd message in a way that breaks the regex,
// this fixture-driven test catches it the moment the fixture is refreshed.
func TestCrossCwdRe_RealCapture(t *testing.T) {
	raw, err := os.ReadFile("testdata/cross-cwd-tty-capture.bin")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	// Strip the script(1) header line ("Script started on ...") and the
	// trailing "Script done on ..." footer — they're added by script, not by
	// claude, and shouldn't bias the regex match.
	if i := bytes.IndexByte(raw, '\n'); i >= 0 {
		raw = raw[i+1:]
	}
	if i := bytes.LastIndex(raw, []byte("\nScript done on ")); i >= 0 {
		raw = raw[:i]
	}

	dest, uuid, ok := detectCrossCwd(raw)
	if !ok {
		t.Fatal("detectCrossCwd: expected match on real PTY capture")
	}
	// Values come from the captured session; if you re-record the fixture,
	// update these expectations to match.
	if dest != "/home/tom/src/fnclaude@fnrhombus" {
		t.Errorf("dest: got %q", dest)
	}
	if uuid != "22d4b53f-265f-4455-9e85-2e1afed6244b" {
		t.Errorf("uuid: got %q", uuid)
	}
}

// Real-world capture: claude's TUI renders the "different directory" preamble
// using cursor-right escapes between words (e.g. `\x1b[1C`) instead of spaces,
// so the preamble text is never plain ASCII in the PTY stream. The cd-and-resume
// line, by contrast, is rendered as plain ASCII. The regex anchors on what
// actually survives. This synthetic test stays as defense-in-depth alongside
// the fixture-driven TestCrossCwdRe_RealCapture above.
func TestCrossCwdRe_TUIWithCursorEscapesBetweenWords(t *testing.T) {
	// Faithful reproduction of a real captured ring tail. The preamble has
	// `\x1b[1C` instead of spaces; the cd line is plain ASCII; between them
	// there is `\x1b[K\r\x1b[1C\x1b[1B` line-erase/CR/cursor-move goo.
	const tuiCapture = "This\x1b[1Cconversation\x1b[1Cis\x1b[1Cfrom\x1b[1Ca\x1b[1Cdifferent\x1b[1Cdirectory.\r\x1b[1B\x1b[K\rTo resume, run:\x1b[K\r\x1b[1C\x1b[1Bcd /home/tom/src/fnclaude@fnrhombus && claude --resume 22d4b53f-265f-4455-9e85-2e1afed6244b\x1b[K"

	dest, uuid, ok := detectCrossCwd([]byte(tuiCapture))
	if !ok {
		t.Fatal("detectCrossCwd: expected match on TUI capture")
	}
	if dest != "/home/tom/src/fnclaude@fnrhombus" {
		t.Errorf("dest: got %q", dest)
	}
	if uuid != "22d4b53f-265f-4455-9e85-2e1afed6244b" {
		t.Errorf("uuid: got %q", uuid)
	}
}

func TestCrossCwdRe_MultipleMatches_LastWins(t *testing.T) {
	// Embed two messages; the last one should win.
	two := claudeExactMessage + "\n" + `This conversation is from a different directory.

To resume, run:
  cd /home/tom/src/dots@rhombu5 && claude --resume aaaabbbb-1111-2222-3333-ccccddddeeee

(Command copied to clipboard)`

	dest, uuid, ok := detectCrossCwd([]byte(two))
	if !ok {
		t.Fatal("detectCrossCwd: expected match")
	}
	if dest != "/home/tom/src/dots@rhombu5" {
		t.Errorf("dest: got %q, want dots path", dest)
	}
	if uuid != "aaaabbbb-1111-2222-3333-ccccddddeeee" {
		t.Errorf("uuid: got %q", uuid)
	}
}

func TestCrossCwdRe_MessageEmbeddedInLargerOutput(t *testing.T) {
	// Simulate real PTY output: lots of normal output before the message.
	prefix := "=== some normal claude output ===\nthinking...\nDone.\n\n"
	input := prefix + claudeExactMessage
	dest, uuid, ok := detectCrossCwd([]byte(input))
	if !ok {
		t.Fatal("detectCrossCwd: expected match in larger output")
	}
	if dest != "/home/tom/src/arch-setup@fnrhombus" {
		t.Errorf("dest: got %q", dest)
	}
	if uuid != "68aa15ae-af23-4c7a-b59f-5cee07c61790" {
		t.Errorf("uuid: got %q", uuid)
	}
}

// ── reconstructArgv tests ─────────────────────────────────────────────────

// reconstructArgv is a pure function; test all spec examples plus edge cases.

type reconstructCase struct {
	name     string
	origArgs []string // os.Args[1:]
	dest     string
	uuid     string
	want     []string
}

var reconstructCases = []reconstructCase{
	{
		name:     "no args",
		origArgs: []string{},
		dest:     "/dest/dir",
		uuid:     "68aa15ae-af23-4c7a-b59f-5cee07c61790",
		want:     []string{"/dest/dir", "--resume", "68aa15ae-af23-4c7a-b59f-5cee07c61790"},
	},
	{
		name:     "single path",
		origArgs: []string{"src/"},
		dest:     "/dest/dir",
		uuid:     "68aa15ae-af23-4c7a-b59f-5cee07c61790",
		want:     []string{"/dest/dir", "--resume", "68aa15ae-af23-4c7a-b59f-5cee07c61790"},
	},
	{
		name:     "two paths replaced by single dest",
		origArgs: []string{"src/", "extra/"},
		dest:     "/dest/dir",
		uuid:     "68aa15ae-af23-4c7a-b59f-5cee07c61790",
		want:     []string{"/dest/dir", "--resume", "68aa15ae-af23-4c7a-b59f-5cee07c61790"},
	},
	{
		name:     "model preserved",
		origArgs: []string{"opus", "src/"},
		dest:     "/dest/dir",
		uuid:     "68aa15ae-af23-4c7a-b59f-5cee07c61790",
		want:     []string{"opus", "/dest/dir", "--resume", "68aa15ae-af23-4c7a-b59f-5cee07c61790"},
	},
	{
		name:     "model and effort preserved, flags preserved",
		origArgs: []string{"opus", "max", "src/", "-V"},
		dest:     "/dest/dir",
		uuid:     "68aa15ae-af23-4c7a-b59f-5cee07c61790",
		want:     []string{"opus", "max", "/dest/dir", "--resume", "68aa15ae-af23-4c7a-b59f-5cee07c61790", "-V"},
	},
	{
		name:     "path then flags",
		origArgs: []string{"src/", "--model", "sonnet", "-V"},
		dest:     "/dest/dir",
		uuid:     "68aa15ae-af23-4c7a-b59f-5cee07c61790",
		want:     []string{"/dest/dir", "--resume", "68aa15ae-af23-4c7a-b59f-5cee07c61790", "--model", "sonnet", "-V"},
	},
	{
		name:     "model with path and also flag",
		origArgs: []string{"opus", "src/", "-A", "docs/"},
		dest:     "/dest/dir",
		uuid:     "68aa15ae-af23-4c7a-b59f-5cee07c61790",
		want:     []string{"opus", "/dest/dir", "--resume", "68aa15ae-af23-4c7a-b59f-5cee07c61790", "-A", "docs/"},
	},
	{
		name:     "model effort path and flag",
		origArgs: []string{"sonnet", "high", "/some/path", "--verbose"},
		dest:     "/dest/dir",
		uuid:     "68aa15ae-af23-4c7a-b59f-5cee07c61790",
		want:     []string{"sonnet", "high", "/dest/dir", "--resume", "68aa15ae-af23-4c7a-b59f-5cee07c61790", "--verbose"},
	},
	{
		name:     "flags only (no path args)",
		origArgs: []string{"--verbose"},
		dest:     "/dest/dir",
		uuid:     "68aa15ae-af23-4c7a-b59f-5cee07c61790",
		want:     []string{"/dest/dir", "--resume", "68aa15ae-af23-4c7a-b59f-5cee07c61790", "--verbose"},
	},
	{
		name:     "model and effort only (no paths, flags later)",
		origArgs: []string{"haiku", "low", "--verbose"},
		dest:     "/dest/dir",
		uuid:     "68aa15ae-af23-4c7a-b59f-5cee07c61790",
		want:     []string{"haiku", "low", "/dest/dir", "--resume", "68aa15ae-af23-4c7a-b59f-5cee07c61790", "--verbose"},
	},
	{
		name:     "multiple magic words then multiple paths",
		origArgs: []string{"opus", "xhigh", "path1/", "path2/", "--flag"},
		dest:     "/dest/dir",
		uuid:     "68aa15ae-af23-4c7a-b59f-5cee07c61790",
		want:     []string{"opus", "xhigh", "/dest/dir", "--resume", "68aa15ae-af23-4c7a-b59f-5cee07c61790", "--flag"},
	},
}

func TestReconstructArgv(t *testing.T) {
	for _, tc := range reconstructCases {
		t.Run(tc.name, func(t *testing.T) {
			got := reconstructArgv(tc.origArgs, tc.dest, tc.uuid)
			if len(got) != len(tc.want) {
				t.Errorf("len: got %d, want %d\n  got:  %v\n  want: %v", len(got), len(tc.want), got, tc.want)
				return
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("[%d]: got %q, want %q\n  full got:  %v\n  full want: %v", i, got[i], tc.want[i], got, tc.want)
				}
			}
		})
	}
}

// ── ringBuffer tests ───────────────────────────────────────────────────────

func TestRingBuffer_SmallWrite(t *testing.T) {
	r := newRingBuffer(16)
	_, _ = r.Write([]byte("hello"))
	if string(r.Bytes()) != "hello" {
		t.Errorf("got %q, want %q", r.Bytes(), "hello")
	}
}

func TestRingBuffer_ExactCapacity(t *testing.T) {
	r := newRingBuffer(5)
	_, _ = r.Write([]byte("12345"))
	if string(r.Bytes()) != "12345" {
		t.Errorf("got %q", r.Bytes())
	}
}

func TestRingBuffer_Overflow(t *testing.T) {
	r := newRingBuffer(5)
	_, _ = r.Write([]byte("1234567890"))
	// Should keep the last 5 bytes.
	if string(r.Bytes()) != "67890" {
		t.Errorf("got %q, want %q", r.Bytes(), "67890")
	}
}

func TestRingBuffer_MultipleWrites(t *testing.T) {
	r := newRingBuffer(6)
	_, _ = r.Write([]byte("abc"))
	_, _ = r.Write([]byte("def"))
	if string(r.Bytes()) != "abcdef" {
		t.Errorf("got %q", r.Bytes())
	}
}

func TestRingBuffer_MultipleWritesOverflow(t *testing.T) {
	r := newRingBuffer(4)
	_, _ = r.Write([]byte("ab"))
	_, _ = r.Write([]byte("cdef")) // total 6, cap 4 → keep "cdef"
	if string(r.Bytes()) != "cdef" {
		t.Errorf("got %q, want %q", r.Bytes(), "cdef")
	}
}

func TestRingBuffer_Empty(t *testing.T) {
	r := newRingBuffer(8)
	if len(r.Bytes()) != 0 {
		t.Errorf("expected empty, got %q", r.Bytes())
	}
}
