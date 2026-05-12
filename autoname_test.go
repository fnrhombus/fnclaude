package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

// ── shouldAutoName ─────────────────────────────────────────────────────────

func TestShouldAutoName(t *testing.T) {
	cases := []struct {
		name        string
		passthrough []string
		want        bool
	}{
		// ── should fire ───────────────────────────────────────────────────────
		{
			name:        "bare dash-dash with prompt",
			passthrough: []string{"--", "fix the login bug"},
			want:        true,
		},
		{
			name:        "flags before dash-dash then prompt",
			passthrough: []string{"--verbose", "--", "add dark mode"},
			want:        true,
		},
		{
			name:        "multiple tokens after dash-dash — only first used but still fires",
			passthrough: []string{"--", "refactor auth", "extra"},
			want:        true,
		},

		// ── should NOT fire: no dash-dash ─────────────────────────────────────
		{
			name:        "no dash-dash",
			passthrough: []string{"--verbose", "something"},
			want:        false,
		},
		{
			name:        "empty",
			passthrough: []string{},
			want:        false,
		},

		// ── should NOT fire: dash-dash but no prompt after ─────────────────────
		{
			name:        "dash-dash at end, no prompt",
			passthrough: []string{"--"},
			want:        false,
		},
		{
			name:        "dash-dash followed only by empty strings",
			passthrough: []string{"--", "", ""},
			want:        false,
		},

		// ── should NOT fire: --name already present ────────────────────────────
		{
			name:        "--name bare",
			passthrough: []string{"--name", "foo", "--", "prompt"},
			want:        false,
		},
		{
			name:        "--name=val",
			passthrough: []string{"--name=mysession", "--", "prompt"},
			want:        false,
		},
		{
			name:        "-n bare",
			passthrough: []string{"-n", "foo", "--", "prompt"},
			want:        false,
		},
		{
			name:        "-n=val",
			passthrough: []string{"-n=foo", "--", "prompt"},
			want:        false,
		},

		// ── should NOT fire: --print ───────────────────────────────────────────
		{
			name:        "-p",
			passthrough: []string{"-p", "--", "prompt"},
			want:        false,
		},
		{
			name:        "--print",
			passthrough: []string{"--print", "--", "prompt"},
			want:        false,
		},

		// ── should NOT fire: --resume ──────────────────────────────────────────
		{
			name:        "-r bare",
			passthrough: []string{"-r", "--", "prompt"},
			want:        false,
		},
		{
			name:        "--resume bare",
			passthrough: []string{"--resume", "--", "prompt"},
			want:        false,
		},
		{
			name:        "-r=val",
			passthrough: []string{"-r=abc123", "--", "prompt"},
			want:        false,
		},
		{
			name:        "--resume=val",
			passthrough: []string{"--resume=abc123", "--", "prompt"},
			want:        false,
		},

		// ── should NOT fire: --continue ────────────────────────────────────────
		{
			name:        "-c",
			passthrough: []string{"-c", "--", "prompt"},
			want:        false,
		},
		{
			name:        "--continue",
			passthrough: []string{"--continue", "--", "prompt"},
			want:        false,
		},

		// ── should NOT fire: --from-pr / -P ───────────────────────────────────
		{
			name:        "--from-pr bare",
			passthrough: []string{"--from-pr", "--", "prompt"},
			want:        false,
		},
		{
			name:        "--from-pr=val",
			passthrough: []string{"--from-pr=123", "--", "prompt"},
			want:        false,
		},
		{
			name:        "-P bare",
			passthrough: []string{"-P", "--", "prompt"},
			want:        false,
		},
		{
			name:        "-P=val",
			passthrough: []string{"-P=123", "--", "prompt"},
			want:        false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldAutoName(tc.passthrough)
			if got != tc.want {
				t.Errorf("shouldAutoName(%v) = %v, want %v", tc.passthrough, got, tc.want)
			}
		})
	}
}

// ── heuristicName ─────────────────────────────────────────────────────────

func TestHeuristicName(t *testing.T) {
	cases := []struct {
		name   string
		prompt string
		want   string
	}{
		{
			name:   "simple three words",
			prompt: "fix login bug",
			want:   "fix-login-bug",
		},
		{
			name:   "stops stripped",
			prompt: "please fix the login bug now",
			want:   "fix-login-bug",
		},
		{
			name:   "exactly three content words",
			prompt: "add dark mode",
			want:   "add-dark-mode",
		},
		{
			name:   "more than three content words — take first three",
			prompt: "refactor the authentication module completely",
			want:   "refactor-authentication-module",
		},
		{
			name:   "punctuation stripped per word",
			prompt: "fix! login. bug?",
			want:   "fix-login-bug",
		},
		{
			name:   "mixed case lowercased",
			prompt: "Add Dark Mode",
			want:   "add-dark-mode",
		},
		{
			name:   "all stop words",
			prompt: "please do the thing for a",
			want:   "thing",
		},
		{
			name:   "empty prompt",
			prompt: "",
			want:   "session",
		},
		{
			name:   "punctuation only",
			prompt: "!!! ???",
			want:   "session",
		},
		{
			name:   "single word",
			prompt: "refactor",
			want:   "refactor",
		},
		{
			name:   "two words",
			prompt: "fix bug",
			want:   "fix-bug",
		},
		{
			name:   "stop words with no content",
			prompt: "the a an",
			want:   "session",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := heuristicName(tc.prompt)
			if got != tc.want {
				t.Errorf("heuristicName(%q) = %q, want %q", tc.prompt, got, tc.want)
			}
		})
	}
}

// ── sanitizeName ──────────────────────────────────────────────────────────

func TestSanitizeName(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "already clean",
			raw:  "fix-login-bug",
			want: "fix-login-bug",
		},
		{
			name: "mixed case",
			raw:  "Fix-Login-Bug",
			want: "fix-login-bug",
		},
		{
			name: "leading Label: prefix — colon and space become dash, label is a segment",
			raw:  "Label: fix-login-bug",
			want: "label-fix-login",
		},
		{
			name: "trailing dot",
			raw:  "fix-login-bug.",
			want: "fix-login-bug",
		},
		{
			name: "emoji stripped",
			raw:  "🔧 fix-login-bug 🐛",
			want: "fix-login-bug",
		},
		{
			name: "spaces become dashes",
			raw:  "fix login bug",
			want: "fix-login-bug",
		},
		{
			name: "more than three segments truncated",
			raw:  "fix-the-login-bug",
			want: "fix-the-login",
		},
		{
			name: "consecutive dashes collapsed",
			raw:  "fix--login---bug",
			want: "fix-login-bug",
		},
		{
			name: "leading and trailing dashes trimmed",
			raw:  "-fix-login-bug-",
			want: "fix-login-bug",
		},
		{
			name: "quoted output",
			raw:  `"fix-login-bug"`,
			want: "fix-login-bug",
		},
		{
			name: "empty input",
			raw:  "",
			want: "",
		},
		{
			name: "whitespace only",
			raw:  "   ",
			want: "",
		},
		{
			name: "only special chars",
			raw:  "!!??##",
			want: "",
		},
		{
			name: "mixed whitespace and special",
			raw:  "  Add Dark Mode  ",
			want: "add-dark-mode",
		},
		{
			name: "explanation text after label",
			raw:  "add-dark-mode (adds a dark mode toggle)",
			want: "add-dark-mode",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeName(tc.raw)
			if got != tc.want {
				t.Errorf("sanitizeName(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

// ── generateName ──────────────────────────────────────────────────────────

// stderrCapture replaces os.Stderr-style writes via a pipe trick.
// Instead, generateName accepts *os.File so we use a temp file in tests.
func tempStderrFile(t *testing.T) *os.File {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "stderr")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

func readStderrFile(t *testing.T, f *os.File) string {
	t.Helper()
	if _, err := f.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(f); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestGenerateName_HappyPath(t *testing.T) {
	cfg := defaultConfig().Name
	apiKey := "test-key"

	llmFn := func(_ context.Context, model, prompt string) (string, error) {
		return "fix-login-bug", nil
	}

	stderr := tempStderrFile(t)
	got := generateName("fix the login bug", cfg, apiKey, llmFn, stderr)
	if got != "fix-login-bug" {
		t.Errorf("got %q, want %q", got, "fix-login-bug")
	}
	// No warning expected.
	if s := readStderrFile(t, stderr); s != "" {
		t.Errorf("unexpected stderr: %q", s)
	}
}

func TestGenerateName_LLMOutputSanitized(t *testing.T) {
	cfg := defaultConfig().Name
	apiKey := "test-key"

	llmFn := func(_ context.Context, model, prompt string) (string, error) {
		return "Fix Login Bug!!!", nil
	}

	stderr := tempStderrFile(t)
	got := generateName("fix the login bug", cfg, apiKey, llmFn, stderr)
	if got != "fix-login-bug" {
		t.Errorf("got %q, want %q", got, "fix-login-bug")
	}
}

func TestGenerateName_Timeout_FallsBackToHeuristic(t *testing.T) {
	cfg := defaultConfig().Name
	cfg.Timeout = 10 * time.Millisecond
	apiKey := "test-key"

	llmFn := func(ctx context.Context, model, prompt string) (string, error) {
		// Simulate a timeout by checking the context.
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(500 * time.Millisecond):
			return "should-not-reach", nil
		}
	}

	stderr := tempStderrFile(t)
	got := generateName("fix the login bug", cfg, apiKey, llmFn, stderr)
	// Should fall back to heuristic (no stop words, so: fix-login-bug)
	if got != "fix-login-bug" {
		t.Errorf("got %q, want %q", got, "fix-login-bug")
	}
	// No warning about missing key.
	if s := readStderrFile(t, stderr); s != "" {
		t.Errorf("unexpected stderr: %q", s)
	}
}

func TestGenerateName_APIError_FallsBackToHeuristic(t *testing.T) {
	cfg := defaultConfig().Name
	apiKey := "test-key"

	llmFn := func(_ context.Context, model, prompt string) (string, error) {
		return "", errors.New("401 Unauthorized")
	}

	stderr := tempStderrFile(t)
	got := generateName("add dark mode", cfg, apiKey, llmFn, stderr)
	if got != "add-dark-mode" {
		t.Errorf("got %q, want %q", got, "add-dark-mode")
	}
	// API errors are silent.
	if s := readStderrFile(t, stderr); s != "" {
		t.Errorf("unexpected stderr output: %q", s)
	}
}

func TestGenerateName_EmptyLLMOutput_FallsBackToHeuristic(t *testing.T) {
	cfg := defaultConfig().Name
	apiKey := "test-key"

	llmFn := func(_ context.Context, model, prompt string) (string, error) {
		return "!!!", nil // sanitizes to ""
	}

	stderr := tempStderrFile(t)
	got := generateName("refactor auth", cfg, apiKey, llmFn, stderr)
	if got != "refactor-auth" {
		t.Errorf("got %q, want %q", got, "refactor-auth")
	}
}

func TestGenerateName_MissingAPIKey_WarnsAndFallsBack(t *testing.T) {
	cfg := defaultConfig().Name
	apiKey := ""

	stderr := tempStderrFile(t)
	got := generateName("add dark mode", cfg, apiKey, nil, stderr)
	if got != "add-dark-mode" {
		t.Errorf("got %q, want %q", got, "add-dark-mode")
	}
	s := readStderrFile(t, stderr)
	if !strings.Contains(s, "ANTHROPIC_API_KEY not set") {
		t.Errorf("expected API key warning on stderr, got: %q", s)
	}
}

func TestGenerateName_MissingAPIKey_QuietMode_NoWarning(t *testing.T) {
	cfg := defaultConfig().Name
	cfg.QuietMissingAPIKey = true
	apiKey := ""

	stderr := tempStderrFile(t)
	got := generateName("add dark mode", cfg, apiKey, nil, stderr)
	if got != "add-dark-mode" {
		t.Errorf("got %q, want %q", got, "add-dark-mode")
	}
	if s := readStderrFile(t, stderr); s != "" {
		t.Errorf("expected no output on stderr in quiet mode, got: %q", s)
	}
}
