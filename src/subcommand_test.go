package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseArgs_Subcommand_ResumeAlone(t *testing.T) {
	a, err := parseArgs([]string{"resume"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a.Passthrough, []string{"--resume"}) {
		t.Errorf("Passthrough: got %v, want [--resume]", a.Passthrough)
	}
	if a.CWD != noopDir {
		t.Errorf("CWD: got %q, want noop", a.CWD)
	}
}

func TestParseArgs_Subcommand_ResShorthand(t *testing.T) {
	a, err := parseArgs([]string{"res"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a.Passthrough, []string{"--resume"}) {
		t.Errorf("Passthrough: got %v, want [--resume]", a.Passthrough)
	}
}

func TestParseArgs_Subcommand_ContinueAlone(t *testing.T) {
	a, err := parseArgs([]string{"continue"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a.Passthrough, []string{"--continue"}) {
		t.Errorf("Passthrough: got %v, want [--continue]", a.Passthrough)
	}
}

func TestParseArgs_Subcommand_ConShorthand(t *testing.T) {
	a, err := parseArgs([]string{"con"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a.Passthrough, []string{"--continue"}) {
		t.Errorf("Passthrough: got %v, want [--continue]", a.Passthrough)
	}
}

// Order-agnostic: subcommand BEFORE model/effort
func TestParseArgs_Subcommand_BeforeModelEffort(t *testing.T) {
	a, err := parseArgs([]string{"resume", "opus", "xhigh"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--resume", "--model", "opus", "--effort", "xhigh"}
	if !reflect.DeepEqual(a.Passthrough, want) {
		t.Errorf("Passthrough: got %v, want %v", a.Passthrough, want)
	}
	if a.CWD != noopDir {
		t.Errorf("CWD: got %q, want noop", a.CWD)
	}
}

// Order-agnostic: subcommand AFTER model/effort
func TestParseArgs_Subcommand_AfterModelEffort(t *testing.T) {
	a, err := parseArgs([]string{"opus", "xhigh", "resume"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--resume", "--model", "opus", "--effort", "xhigh"}
	if !reflect.DeepEqual(a.Passthrough, want) {
		t.Errorf("Passthrough: got %v, want %v", a.Passthrough, want)
	}
}

// Order-agnostic: subcommand BETWEEN model and effort
func TestParseArgs_Subcommand_BetweenModelAndEffort(t *testing.T) {
	a, err := parseArgs([]string{"opus", "resume", "xhigh"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--resume", "--model", "opus", "--effort", "xhigh"}
	if !reflect.DeepEqual(a.Passthrough, want) {
		t.Errorf("Passthrough: got %v, want %v", a.Passthrough, want)
	}
}

func TestParseArgs_Subcommand_WithCwdBefore(t *testing.T) {
	a, err := parseArgs([]string{"/proj/foo", "resume"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if a.CWD != "/proj/foo" {
		t.Errorf("CWD: got %q, want /proj/foo", a.CWD)
	}
	if !reflect.DeepEqual(a.Passthrough, []string{"--resume"}) {
		t.Errorf("Passthrough: got %v, want [--resume]", a.Passthrough)
	}
}

func TestParseArgs_Subcommand_WithCwdAfter(t *testing.T) {
	a, err := parseArgs([]string{"resume", "/proj/foo"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if a.CWD != "/proj/foo" {
		t.Errorf("CWD: got %q, want /proj/foo", a.CWD)
	}
	if !reflect.DeepEqual(a.Passthrough, []string{"--resume"}) {
		t.Errorf("Passthrough: got %v, want [--resume]", a.Passthrough)
	}
}

// `./resume` is a literal path, not the subcommand
func TestParseArgs_Subcommand_DotPrefixedShadowsSubcommand(t *testing.T) {
	a, err := parseArgs([]string{"./resume"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if a.CWD != "./resume" {
		t.Errorf("CWD: got %q, want ./resume", a.CWD)
	}
	if len(a.Passthrough) != 0 {
		t.Errorf("Passthrough: got %v, want empty", a.Passthrough)
	}
}

func TestParseArgs_Subcommand_TwoSubcommandsError(t *testing.T) {
	_, err := parseArgs([]string{"resume", "continue"}, testHome)
	if err == nil {
		t.Fatal("expected error for two subcommands, got nil")
	}
	if !strings.Contains(err.Error(), "subcommand") {
		t.Errorf("error should mention 'subcommand', got: %v", err)
	}
}

func TestParseArgs_Subcommand_ResAndConTogetherError(t *testing.T) {
	_, err := parseArgs([]string{"res", "con"}, testHome)
	if err == nil {
		t.Fatal("expected error for two subcommand shorthands")
	}
}

// ── fork subcommand ──────────────────────────────────────────────────────────
// `fork` expands to BOTH --resume and --fork-session, so the picker opens and
// any selected session is forked (gets a new session id) on resume.

func TestParseArgs_Subcommand_ForkAlone(t *testing.T) {
	a, err := parseArgs([]string{"fork"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--resume", "--fork-session"}
	if !reflect.DeepEqual(a.Passthrough, want) {
		t.Errorf("Passthrough: got %v, want %v", a.Passthrough, want)
	}
	if a.CWD != noopDir {
		t.Errorf("CWD: got %q, want noop", a.CWD)
	}
}

func TestParseArgs_Subcommand_FkShorthand(t *testing.T) {
	a, err := parseArgs([]string{"fk"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--resume", "--fork-session"}
	if !reflect.DeepEqual(a.Passthrough, want) {
		t.Errorf("Passthrough: got %v, want %v", a.Passthrough, want)
	}
}

func TestParseArgs_Subcommand_ForkWithModelEffort(t *testing.T) {
	a, err := parseArgs([]string{"fork", "opus", "xhigh"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--resume", "--fork-session", "--model", "opus", "--effort", "xhigh"}
	if !reflect.DeepEqual(a.Passthrough, want) {
		t.Errorf("Passthrough: got %v, want %v", a.Passthrough, want)
	}
}

// Order-agnostic: model+effort BEFORE fork.
func TestParseArgs_Subcommand_ModelEffortBeforeFork(t *testing.T) {
	a, err := parseArgs([]string{"opus", "xhigh", "fork"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--resume", "--fork-session", "--model", "opus", "--effort", "xhigh"}
	if !reflect.DeepEqual(a.Passthrough, want) {
		t.Errorf("Passthrough: got %v, want %v", a.Passthrough, want)
	}
}

func TestParseArgs_Subcommand_ForkWithCwd(t *testing.T) {
	a, err := parseArgs([]string{"/proj/foo", "fork"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if a.CWD != "/proj/foo" {
		t.Errorf("CWD: got %q, want /proj/foo", a.CWD)
	}
	want := []string{"--resume", "--fork-session"}
	if !reflect.DeepEqual(a.Passthrough, want) {
		t.Errorf("Passthrough: got %v, want %v", a.Passthrough, want)
	}
}

// `./fork` is a literal path (consistent with `./resume`).
func TestParseArgs_Subcommand_DotPrefixedShadowsFork(t *testing.T) {
	a, err := parseArgs([]string{"./fork"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if a.CWD != "./fork" {
		t.Errorf("CWD: got %q, want ./fork", a.CWD)
	}
	if len(a.Passthrough) != 0 {
		t.Errorf("Passthrough: got %v, want empty", a.Passthrough)
	}
}

// `fork` collides with any other subcommand — same one-subcommand-only rule.
func TestParseArgs_Subcommand_ForkAndResumeError(t *testing.T) {
	_, err := parseArgs([]string{"fork", "resume"}, testHome)
	if err == nil {
		t.Fatal("expected error for fork + resume")
	}
	if !strings.Contains(err.Error(), "subcommand") {
		t.Errorf("error should mention 'subcommand', got: %v", err)
	}
}

func TestParseArgs_Subcommand_ForkAndContinueError(t *testing.T) {
	_, err := parseArgs([]string{"fork", "continue"}, testHome)
	if err == nil {
		t.Fatal("expected error for fork + continue")
	}
}

func TestParseArgs_Subcommand_ForkSubcommandBeforeDashDash(t *testing.T) {
	a, err := parseArgs([]string{"fork", "--", "the prompt"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--resume", "--fork-session", "--", "the prompt"}
	if !reflect.DeepEqual(a.Passthrough, want) {
		t.Errorf("Passthrough: got %v, want %v", a.Passthrough, want)
	}
}

// Subcommand goes before -- in passthrough so claude reads it as a flag,
// not as part of the prompt args.
func TestParseArgs_Subcommand_WithDashDashPrompt(t *testing.T) {
	a, err := parseArgs([]string{"resume", "--", "the prompt"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--resume", "--", "the prompt"}
	if !reflect.DeepEqual(a.Passthrough, want) {
		t.Errorf("Passthrough: got %v, want %v", a.Passthrough, want)
	}
}

// After --verbose triggers flag mode, `resume` is no longer a subcommand.
func TestParseArgs_Subcommand_AfterFlagsIsLiteral(t *testing.T) {
	a, err := parseArgs([]string{"--verbose", "resume"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	// `resume` should be in passthrough as a plain token, not become --resume.
	for _, t2 := range a.Passthrough {
		if t2 == "--resume" {
			t.Errorf("Passthrough unexpectedly contains --resume: %v", a.Passthrough)
		}
	}
	foundLiteral := false
	for _, t2 := range a.Passthrough {
		if t2 == "resume" {
			foundLiteral = true
			break
		}
	}
	if !foundLiteral {
		t.Errorf("expected literal 'resume' in Passthrough, got %v", a.Passthrough)
	}
}
