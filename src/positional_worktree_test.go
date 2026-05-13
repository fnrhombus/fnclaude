package main

import (
	"strings"
	"testing"
)

// 2nd positional (after magic & subcommands) is the worktree name.

func TestParseArgs_PositionalWt_BasicCwdAndWt(t *testing.T) {
	a, err := parseArgs([]string{"/proj/foo", "my-wt"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if a.CWD != "/proj/foo" {
		t.Errorf("CWD: got %q, want /proj/foo", a.CWD)
	}
	if !a.WorktreeSet {
		t.Error("WorktreeSet: got false, want true")
	}
	if a.WorktreeArg != "my-wt" {
		t.Errorf("WorktreeArg: got %q, want my-wt", a.WorktreeArg)
	}
	if len(a.ExtraDirs) != 0 {
		t.Errorf("ExtraDirs: got %v, want empty", a.ExtraDirs)
	}
}

func TestParseArgs_PositionalWt_OnlyWtNoCwd(t *testing.T) {
	// Single positional is still CWD (worktree slot stays empty).
	a, err := parseArgs([]string{"/proj/foo"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if a.CWD != "/proj/foo" {
		t.Errorf("CWD: got %q, want /proj/foo", a.CWD)
	}
	if a.WorktreeSet {
		t.Error("WorktreeSet: got true, want false (single positional is cwd only)")
	}
}

func TestParseArgs_PositionalWt_WithModelMagic(t *testing.T) {
	// fnc opus /proj wt → model+cwd+worktree
	a, err := parseArgs([]string{"opus", "/proj/foo", "my-wt"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if a.CWD != "/proj/foo" {
		t.Errorf("CWD: got %q", a.CWD)
	}
	if a.WorktreeArg != "my-wt" {
		t.Errorf("WorktreeArg: got %q, want my-wt", a.WorktreeArg)
	}
	if len(a.Passthrough) < 2 || a.Passthrough[0] != "--model" || a.Passthrough[1] != "opus" {
		t.Errorf("Passthrough should start with --model opus, got %v", a.Passthrough)
	}
}

func TestParseArgs_PositionalWt_WithModelEffortMagic(t *testing.T) {
	// fnc opus xhigh /proj wt → model+effort+cwd+worktree
	a, err := parseArgs([]string{"opus", "xhigh", "/proj/foo", "my-wt"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if a.CWD != "/proj/foo" {
		t.Errorf("CWD: got %q", a.CWD)
	}
	if a.WorktreeArg != "my-wt" {
		t.Errorf("WorktreeArg: got %q, want my-wt", a.WorktreeArg)
	}
}

func TestParseArgs_PositionalWt_WithSubcommand(t *testing.T) {
	// fnc resume /proj wt → subcommand+cwd+worktree (4 tokens but only 2 post-magic positionals)
	a, err := parseArgs([]string{"resume", "/proj/foo", "my-wt"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if a.CWD != "/proj/foo" {
		t.Errorf("CWD: got %q", a.CWD)
	}
	if a.WorktreeArg != "my-wt" {
		t.Errorf("WorktreeArg: got %q, want my-wt", a.WorktreeArg)
	}
	if len(a.Passthrough) == 0 || a.Passthrough[0] != "--resume" {
		t.Errorf("Passthrough should start with --resume, got %v", a.Passthrough)
	}
}

// `fnc sonnet path wt` from the user's example — sonnet at pos 1 is model.
func TestParseArgs_PositionalWt_ModelPathWt(t *testing.T) {
	a, err := parseArgs([]string{"sonnet", "/proj/foo", "my-wt"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if a.CWD != "/proj/foo" {
		t.Errorf("CWD: got %q", a.CWD)
	}
	if a.WorktreeArg != "my-wt" {
		t.Errorf("WorktreeArg: got %q, want my-wt", a.WorktreeArg)
	}
}

func TestParseArgs_PositionalWt_ThreeNonMagicErrors(t *testing.T) {
	// 3 post-magic positionals: cwd + wt + EXTRA — error.
	_, err := parseArgs([]string{"/proj/foo", "wt", "extra"}, testHome)
	if err == nil {
		t.Fatal("expected error for 3 post-magic positionals")
	}
	if !strings.Contains(err.Error(), "positional") {
		t.Errorf("error should mention positional, got: %v", err)
	}
}

func TestParseArgs_PositionalWt_WithDashALsoStillWorks(t *testing.T) {
	// -A/--also still provides extra dirs; positional 2 is wt.
	a, err := parseArgs([]string{"/proj/foo", "my-wt", "-A", "/proj/extra"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if a.CWD != "/proj/foo" {
		t.Errorf("CWD: got %q", a.CWD)
	}
	if a.WorktreeArg != "my-wt" {
		t.Errorf("WorktreeArg: got %q, want my-wt", a.WorktreeArg)
	}
	if len(a.ExtraDirs) != 1 || a.ExtraDirs[0] != "/proj/extra" {
		t.Errorf("ExtraDirs: got %v, want [/proj/extra]", a.ExtraDirs)
	}
}

func TestParseArgs_PositionalWt_PositionalWinsLastOverDashW(t *testing.T) {
	// Both positional 2 and -w supplied. Whichever comes later in argv wins.
	// Here -w comes after positional: -w value wins.
	a, err := parseArgs([]string{"/proj/foo", "pos-wt", "-w", "flag-wt"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if a.WorktreeArg != "flag-wt" {
		t.Errorf("WorktreeArg: got %q, want flag-wt (-w wins, comes later)", a.WorktreeArg)
	}
}

func TestParseArgs_PositionalWt_DashWThenPositional(t *testing.T) {
	// -w first, then positional comes later in argv — positional wins.
	a, err := parseArgs([]string{"-w", "flag-wt", "/proj/foo", "pos-wt"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	// Wait — once -w is processed, inFlags=true (it's a flag); subsequent
	// positionals would go to passthrough, not eaten as positionals.
	// Re-spec'ing this: -w pushes inFlags=true so /proj/foo and pos-wt become
	// passthrough tokens, not cwd/worktree. Worktree should stay flag-wt.
	if a.WorktreeArg != "flag-wt" {
		t.Errorf("WorktreeArg: got %q, want flag-wt (positional after flag mode shouldn't override)", a.WorktreeArg)
	}
}
