package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

const testHome = "/home/testuser"

// noopDir is the expected fallback cwd.
var noopDir = filepath.Join(testHome, ".claude", "noop")

// ── parseArgs tests ────────────────────────────────────────────────────────

func TestParseArgs_SinglePositional(t *testing.T) {
	a, err := parseArgs([]string{"/proj/foo"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if a.CWD != "/proj/foo" {
		t.Errorf("CWD: got %q, want %q", a.CWD, "/proj/foo")
	}
	if len(a.ExtraDirs) != 0 {
		t.Errorf("ExtraDirs: got %v, want empty", a.ExtraDirs)
	}
}

func TestParseArgs_ThreePositionals(t *testing.T) {
	a, err := parseArgs([]string{"/proj/a", "/proj/b", "/proj/c"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if a.CWD != "/proj/a" {
		t.Errorf("CWD: got %q, want %q", a.CWD, "/proj/a")
	}
	want := []string{"/proj/b", "/proj/c"}
	if len(a.ExtraDirs) != len(want) {
		t.Fatalf("ExtraDirs len: got %d, want %d", len(a.ExtraDirs), len(want))
	}
	for i, d := range want {
		if a.ExtraDirs[i] != d {
			t.Errorf("ExtraDirs[%d]: got %q, want %q", i, a.ExtraDirs[i], d)
		}
	}
}

func TestParseArgs_ZeroPositionals_Fallback(t *testing.T) {
	a, err := parseArgs([]string{}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if a.CWD != noopDir {
		t.Errorf("CWD: got %q, want %q", a.CWD, noopDir)
	}
}

func TestParseArgs_MixedPositionalAndAlso(t *testing.T) {
	a, err := parseArgs([]string{"/p/main", "/p/extra1", "--also", "/p/extra2", "-A", "/p/extra3"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if a.CWD != "/p/main" {
		t.Errorf("CWD: got %q", a.CWD)
	}
	// extra1 comes from positional, extra2 from --also, extra3 from -A
	want := []string{"/p/extra1", "/p/extra2", "/p/extra3"}
	if len(a.ExtraDirs) != len(want) {
		t.Fatalf("ExtraDirs: got %v, want %v", a.ExtraDirs, want)
	}
	for i, d := range want {
		if a.ExtraDirs[i] != d {
			t.Errorf("ExtraDirs[%d]: got %q, want %q", i, a.ExtraDirs[i], d)
		}
	}
}

func TestParseArgs_AlsoEquals(t *testing.T) {
	a, err := parseArgs([]string{"/p/main", "-A=/p/extra"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.ExtraDirs) != 1 || a.ExtraDirs[0] != "/p/extra" {
		t.Errorf("ExtraDirs: got %v", a.ExtraDirs)
	}
}

func TestParseArgs_AlsoLongEquals(t *testing.T) {
	a, err := parseArgs([]string{"/p/main", "--also=/p/extra"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.ExtraDirs) != 1 || a.ExtraDirs[0] != "/p/extra" {
		t.Errorf("ExtraDirs: got %v", a.ExtraDirs)
	}
}

// Missing-value errors.

func TestParseArgs_MissingValue_Also_AtEOF(t *testing.T) {
	_, err := parseArgs([]string{"/p/x", "-A"}, testHome)
	if err == nil {
		t.Fatal("expected error for -A with no value")
	}
}

func TestParseArgs_MissingValue_Also_NextIsFlag(t *testing.T) {
	_, err := parseArgs([]string{"/p/x", "-A", "--some-flag"}, testHome)
	if err == nil {
		t.Fatal("expected error for -A followed by flag")
	}
}

// --setting-sources detection.

func TestParseArgs_SettingSourcesBare(t *testing.T) {
	a, err := parseArgs([]string{"/p/x", "--setting-sources"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if !settingSourcesInPassthrough(a.Passthrough) {
		t.Error("expected settingSourcesInPassthrough to return true")
	}
}

func TestParseArgs_SettingSourcesWithValue(t *testing.T) {
	a, err := parseArgs([]string{"/p/x", "--setting-sources=foo"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if !settingSourcesInPassthrough(a.Passthrough) {
		t.Error("expected settingSourcesInPassthrough to return true")
	}
}

func TestParseArgs_NoSettingSources(t *testing.T) {
	a, err := parseArgs([]string{"/p/x", "--verbose"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if settingSourcesInPassthrough(a.Passthrough) {
		t.Error("expected settingSourcesInPassthrough to return false")
	}
}

// Passthrough preservation and ordering.

func TestParseArgs_PassthroughOrdering(t *testing.T) {
	a, err := parseArgs([]string{"/p/x", "--foo", "--bar", "--baz"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--foo", "--bar", "--baz"}
	if len(a.Passthrough) != len(want) {
		t.Fatalf("Passthrough len: got %d, want %d", len(a.Passthrough), len(want))
	}
	for i, f := range want {
		if a.Passthrough[i] != f {
			t.Errorf("Passthrough[%d]: got %q, want %q", i, a.Passthrough[i], f)
		}
	}
}

// ── Magic positional tests ─────────────────────────────────────────────────
// Magic rule: position 1 may be a model alias; position 2 (only if pos 1
// matched a model) may be an effort level. Effort alone in position 1 is not
// magic — it becomes the cwd.

func TestParseArgs_Magic_ModelThenPath(t *testing.T) {
	// fnc opus ~/p → --model opus, cwd=~/p
	a, err := parseArgs([]string{"opus", "/proj/p"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if a.CWD != "/proj/p" {
		t.Errorf("CWD: got %q", a.CWD)
	}
	if len(a.Passthrough) != 2 || a.Passthrough[0] != "--model" || a.Passthrough[1] != "opus" {
		t.Errorf("Passthrough: got %v, want [--model opus]", a.Passthrough)
	}
}

func TestParseArgs_Magic_ModelEffortPath(t *testing.T) {
	// fnc opus max ~/p → --model opus --effort max, cwd=~/p
	a, err := parseArgs([]string{"opus", "max", "/proj/p"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if a.CWD != "/proj/p" {
		t.Errorf("CWD: got %q", a.CWD)
	}
	want := []string{"--model", "opus", "--effort", "max"}
	if len(a.Passthrough) != len(want) {
		t.Fatalf("Passthrough: got %v, want %v", a.Passthrough, want)
	}
	for i, w := range want {
		if a.Passthrough[i] != w {
			t.Errorf("Passthrough[%d]: got %q, want %q", i, a.Passthrough[i], w)
		}
	}
}

func TestParseArgs_Magic_EffortAloneIsPath(t *testing.T) {
	// fnc max ~/p → cwd='max', extra dir=~/p (effort alone is not magic)
	a, err := parseArgs([]string{"max", "/proj/p"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if a.CWD != "max" {
		t.Errorf("CWD: got %q, want 'max'", a.CWD)
	}
	if len(a.ExtraDirs) != 1 || a.ExtraDirs[0] != "/proj/p" {
		t.Errorf("ExtraDirs: got %v, want [/proj/p]", a.ExtraDirs)
	}
	if len(a.Passthrough) != 0 {
		t.Errorf("Passthrough: got %v, want empty", a.Passthrough)
	}
}

func TestParseArgs_Magic_ModelThenNonEffortBecomesPath(t *testing.T) {
	// fnc opus sonnet ~/p → --model opus, cwd='sonnet', extra dir=~/p
	// (pos 2 'sonnet' is not an effort level → becomes cwd)
	a, err := parseArgs([]string{"opus", "sonnet", "/proj/p"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Passthrough) != 2 || a.Passthrough[0] != "--model" || a.Passthrough[1] != "opus" {
		t.Errorf("Passthrough: got %v, want [--model opus]", a.Passthrough)
	}
	if a.CWD != "sonnet" {
		t.Errorf("CWD: got %q, want 'sonnet'", a.CWD)
	}
	if len(a.ExtraDirs) != 1 || a.ExtraDirs[0] != "/proj/p" {
		t.Errorf("ExtraDirs: got %v, want [/proj/p]", a.ExtraDirs)
	}
}

func TestParseArgs_Magic_ModelEffortThenExtraDirs(t *testing.T) {
	// fnc opus max sonnet ~/p → --model opus --effort max, cwd='sonnet', extra dir=~/p
	a, err := parseArgs([]string{"opus", "max", "sonnet", "/proj/p"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--model", "opus", "--effort", "max"}
	if len(a.Passthrough) != len(want) {
		t.Fatalf("Passthrough: got %v, want %v", a.Passthrough, want)
	}
	for i, w := range want {
		if a.Passthrough[i] != w {
			t.Errorf("Passthrough[%d]: got %q, want %q", i, a.Passthrough[i], w)
		}
	}
	if a.CWD != "sonnet" {
		t.Errorf("CWD: got %q, want 'sonnet'", a.CWD)
	}
	if len(a.ExtraDirs) != 1 || a.ExtraDirs[0] != "/proj/p" {
		t.Errorf("ExtraDirs: got %v, want [/proj/p]", a.ExtraDirs)
	}
}

func TestParseArgs_Magic_ModelOnly_NoPath_FallsBackToNoop(t *testing.T) {
	// fnc opus → --model opus, cwd=~/.claude/noop
	a, err := parseArgs([]string{"opus"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if a.CWD != noopDir {
		t.Errorf("CWD: got %q, want %q", a.CWD, noopDir)
	}
	if len(a.Passthrough) != 2 || a.Passthrough[0] != "--model" || a.Passthrough[1] != "opus" {
		t.Errorf("Passthrough: got %v, want [--model opus]", a.Passthrough)
	}
}

func TestParseArgs_Magic_ModelAndEffort_NoPath_FallsBackToNoop(t *testing.T) {
	// fnc opus max → --model opus --effort max, cwd=~/.claude/noop
	a, err := parseArgs([]string{"opus", "max"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if a.CWD != noopDir {
		t.Errorf("CWD: got %q, want %q", a.CWD, noopDir)
	}
	want := []string{"--model", "opus", "--effort", "max"}
	if len(a.Passthrough) != len(want) {
		t.Fatalf("Passthrough: got %v, want %v", a.Passthrough, want)
	}
	for i, w := range want {
		if a.Passthrough[i] != w {
			t.Errorf("Passthrough[%d]: got %q, want %q", i, a.Passthrough[i], w)
		}
	}
}

func TestParseArgs_Magic_NonModelFirstTurnsOffMagic(t *testing.T) {
	// fnc /proj sonnet → cwd=/proj, extra dir=sonnet (magic off after pos 1 non-match)
	a, err := parseArgs([]string{"/proj/p", "sonnet"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if a.CWD != "/proj/p" {
		t.Errorf("CWD: got %q", a.CWD)
	}
	if len(a.ExtraDirs) != 1 || a.ExtraDirs[0] != "sonnet" {
		t.Errorf("ExtraDirs: got %v, want [sonnet]", a.ExtraDirs)
	}
	if len(a.Passthrough) != 0 {
		t.Errorf("Passthrough: got %v, want empty", a.Passthrough)
	}
}

func TestParseArgs_DotPrefixedPath_NoMagic(t *testing.T) {
	// ./opus is a literal path, not a magic word.
	a, err := parseArgs([]string{"./opus"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if a.CWD != "./opus" {
		t.Errorf("CWD: got %q, want ./opus", a.CWD)
	}
	if len(a.Passthrough) != 0 {
		t.Errorf("Passthrough: got %v, want empty", a.Passthrough)
	}
}

// ── buildArgv tests ────────────────────────────────────────────────────────
// These tests work against a synthetic fs view where we control which files
// "exist". We test argv assembly without actually execing claude.

func TestBuildArgv_NoExtraDirs(t *testing.T) {
	a := Args{
		CWD:         "/p/main",
		Passthrough: []string{"--verbose"},
	}
	argv := buildArgv(a, "/shell", defaultConfig())
	// Default config has auto.dangerously_skip_permissions=false; not injected.
	// withAgentPitfallWarning appends --append-system-prompt at the tail.
	want := []string{"claude", "--verbose", "--append-system-prompt", agentPitfallWarning}
	assertArgv(t, argv, want)
}

func TestBuildArgv_ExtraDirsAbsolute(t *testing.T) {
	// Use a dir that definitely doesn't have .mcp.json or settings.json.
	a := Args{
		CWD:       "/p/main",
		ExtraDirs: []string{"/p/extra"},
	}
	argv := buildArgv(a, "/shell", defaultConfig())
	// Expect --add-dir; no --mcp-config or --settings (files don't exist).
	want := []string{"claude", "--add-dir", "/p/extra", "--append-system-prompt", agentPitfallWarning}
	assertArgv(t, argv, want)
}

func TestBuildArgv_RelativeExtraDirResolved(t *testing.T) {
	a := Args{
		CWD:       "/p/main",
		ExtraDirs: []string{"relative/dir"},
	}
	argv := buildArgv(a, "/shell/cwd", defaultConfig())
	// The relative path should be joined with shellCWD.
	want := []string{"claude", "--add-dir", "/shell/cwd/relative/dir", "--append-system-prompt", agentPitfallWarning}
	assertArgv(t, argv, want)
}

func TestBuildArgv_SettingSourcesSuppressesSettings(t *testing.T) {
	// Even if a settings file existed this confirms the logic path.
	// Since no files exist on disk here, --settings wouldn't appear anyway,
	// but the passthrough token must be present.
	a := Args{
		CWD:         "/p/main",
		ExtraDirs:   []string{"/p/extra"},
		Passthrough: []string{"--setting-sources=user"},
	}
	argv := buildArgv(a, "/shell", defaultConfig())
	// --setting-sources should be in passthrough position, after injected flags.
	assertContains(t, argv, "--setting-sources=user")
	assertNotContains(t, argv, "--settings")
}

func TestBuildArgv_MultipleExtraDir_Order(t *testing.T) {
	a := Args{
		CWD:       "/p/main",
		ExtraDirs: []string{"/p/b", "/p/c"},
	}
	argv := buildArgv(a, "/shell", defaultConfig())
	want := []string{
		"claude",
		"--add-dir", "/p/b",
		"--add-dir", "/p/c",
		"--append-system-prompt", agentPitfallWarning,
	}
	assertArgv(t, argv, want)
}

func TestBuildArgv_AutoSkipPermissions_Injected(t *testing.T) {
	cfg := defaultConfig()
	cfg.Auto.DangerouslySkipPermissions = true
	a := Args{CWD: "/p/main"}
	argv := buildArgv(a, "/shell", cfg)
	assertContains(t, argv, "--dangerously-skip-permissions")
}

func TestBuildArgv_AutoSkipPermissions_SuppressedByNoPermissions(t *testing.T) {
	cfg := defaultConfig()
	cfg.Auto.DangerouslySkipPermissions = true
	a := Args{CWD: "/p/main", NoPermissions: true}
	argv := buildArgv(a, "/shell", cfg)
	assertNotContains(t, argv, "--dangerously-skip-permissions")
}

func TestBuildArgv_ExplicitD_WinsOverNoPermissions(t *testing.T) {
	// -D (translated) puts --dangerously-skip-permissions in passthrough.
	// --no-permissions sets NoPermissions. Explicit -D still wins.
	cfg := defaultConfig()
	a := Args{
		CWD:           "/p/main",
		Passthrough:   []string{"--dangerously-skip-permissions"},
		NoPermissions: true,
	}
	argv := buildArgv(a, "/shell", cfg)
	assertContains(t, argv, "--dangerously-skip-permissions")
}

func TestBuildArgv_NoAutoSkipPermissions_ByDefault(t *testing.T) {
	// Default config has auto.dangerously_skip_permissions=false.
	a := Args{CWD: "/p/main"}
	argv := buildArgv(a, "/shell", defaultConfig())
	assertNotContains(t, argv, "--dangerously-skip-permissions")
}

// ── Auto-IDE injection tests ───────────────────────────────────────────────

func TestBuildArgv_AutoIDE_Always_Injected(t *testing.T) {
	cfg := defaultConfig()
	cfg.Auto.IDE = "always"
	a := Args{CWD: "/p/main"}
	argv := buildArgv(a, "/shell", cfg)
	assertContains(t, argv, "--ide")
}

func TestBuildArgv_AutoIDE_Never_NotInjected(t *testing.T) {
	cfg := defaultConfig()
	cfg.Auto.IDE = "never"
	a := Args{CWD: "/p/main"}
	argv := buildArgv(a, "/shell", cfg)
	assertNotContains(t, argv, "--ide")
}

func TestBuildArgv_AutoIDE_AlreadyInPassthrough_NotDuplicated(t *testing.T) {
	// -I translates to --ide; auto.ide=always should not add a second copy.
	cfg := defaultConfig()
	cfg.Auto.IDE = "always"
	a := Args{
		CWD:         "/p/main",
		Passthrough: []string{"--ide"},
	}
	argv := buildArgv(a, "/shell", cfg)
	count := 0
	for _, tok := range argv {
		if tok == "--ide" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("--ide appears %d times in argv, want exactly 1: %v", count, argv)
	}
}

// ── Auto-TMUX injection tests ──────────────────────────────────────────────

func TestBuildArgv_AutoTmux_Always_Injected(t *testing.T) {
	cfg := defaultConfig()
	cfg.Auto.Tmux = "always"
	a := Args{CWD: "/p/main"}
	argv := buildArgv(a, "/shell", cfg)
	assertContains(t, argv, "--tmux")
}

func TestBuildArgv_AutoTmux_Never_NotInjected(t *testing.T) {
	cfg := defaultConfig()
	cfg.Auto.Tmux = "never"
	a := Args{CWD: "/p/main"}
	argv := buildArgv(a, "/shell", cfg)
	assertNotContains(t, argv, "--tmux")
}

func TestBuildArgv_AutoTmux_Always_SuppressedByNoTmux(t *testing.T) {
	cfg := defaultConfig()
	cfg.Auto.Tmux = "always"
	a := Args{CWD: "/p/main", NoTmux: true}
	argv := buildArgv(a, "/shell", cfg)
	assertNotContains(t, argv, "--tmux")
}

func TestBuildArgv_AutoTmux_Always_AlreadyInPassthrough_NotDuplicated(t *testing.T) {
	cfg := defaultConfig()
	cfg.Auto.Tmux = "always"
	a := Args{
		CWD:         "/p/main",
		Passthrough: []string{"--tmux"},
	}
	argv := buildArgv(a, "/shell", cfg)
	count := 0
	for _, tok := range argv {
		if tok == "--tmux" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("--tmux appears %d times in argv, want exactly 1: %v", count, argv)
	}
}

func TestBuildArgv_AutoTmux_Worktree_NewWorktree(t *testing.T) {
	// -w <name> that didn't match an existing worktree → new worktree being
	// created → --tmux should be injected.
	cfg := defaultConfig()
	cfg.Auto.Tmux = "worktree"
	a := Args{
		CWD:         "/p/main",
		WorktreeSet: true,
		// WorktreeMatched = false (default): -w went through to passthrough.
		Passthrough: []string{"--worktree", "feat"},
	}
	argv := buildArgv(a, "/shell", cfg)
	assertContains(t, argv, "--tmux")
}

func TestBuildArgv_AutoTmux_Worktree_MatchedExisting_NoTmux(t *testing.T) {
	// -w <name> that DID match an existing worktree → cwd swapped, no new
	// worktree → --tmux should NOT be injected.
	cfg := defaultConfig()
	cfg.Auto.Tmux = "worktree"
	a := Args{
		CWD:             "/p/feat",
		WorktreeSet:     true,
		WorktreeMatched: true,
	}
	argv := buildArgv(a, "/shell", cfg)
	assertNotContains(t, argv, "--tmux")
}

func TestBuildArgv_AutoTmux_Worktree_WithoutWorktreeFlag(t *testing.T) {
	cfg := defaultConfig()
	cfg.Auto.Tmux = "worktree"
	a := Args{CWD: "/p/main"}
	argv := buildArgv(a, "/shell", cfg)
	assertNotContains(t, argv, "--tmux")
}

func TestBuildArgv_AutoTmux_Always_ExplicitTNotDuplicated(t *testing.T) {
	// -T translates to --tmux; auto.tmux=always must not add a second.
	cfg := defaultConfig()
	cfg.Auto.Tmux = "always"
	a := Args{
		CWD:         "/p/main",
		Passthrough: []string{"--tmux", "mywin"},
	}
	argv := buildArgv(a, "/shell", cfg)
	count := 0
	for _, tok := range argv {
		if tok == "--tmux" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("--tmux appears %d times in argv, want exactly 1: %v", count, argv)
	}
}

// ── Short-flag translation tests ───────────────────────────────────────────

func TestParseArgs_ShortNoValue_Single(t *testing.T) {
	a, err := parseArgs([]string{"/p/x", "-B"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Passthrough) != 1 || a.Passthrough[0] != "--brief" {
		t.Errorf("Passthrough: got %v, want [--brief]", a.Passthrough)
	}
}

func TestParseArgs_ShortNoValue_AllFlags(t *testing.T) {
	cases := []struct {
		short string
		long  string
	}{
		{"-B", "--brief"},
		{"-C", "--chrome"},
		{"-D", "--dangerously-skip-permissions"},
		{"-F", "--fork-session"},
		{"-I", "--ide"},
		{"-V", "--verbose"},
	}
	for _, tc := range cases {
		a, err := parseArgs([]string{"/p/x", tc.short}, testHome)
		if err != nil {
			t.Fatalf("%s: %v", tc.short, err)
		}
		if len(a.Passthrough) != 1 || a.Passthrough[0] != tc.long {
			t.Errorf("%s: Passthrough got %v, want [%s]", tc.short, a.Passthrough, tc.long)
		}
	}
}

func TestParseArgs_ShortCollapsed_NoValue(t *testing.T) {
	// -BVC → --brief --verbose --chrome
	a, err := parseArgs([]string{"/p/x", "-BVC"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--brief", "--verbose", "--chrome"}
	if len(a.Passthrough) != len(want) {
		t.Fatalf("Passthrough: got %v, want %v", a.Passthrough, want)
	}
	for i, w := range want {
		if a.Passthrough[i] != w {
			t.Errorf("Passthrough[%d]: got %q, want %q", i, a.Passthrough[i], w)
		}
	}
}

func TestParseArgs_ShortRequired_Space(t *testing.T) {
	// -G myagent → --agent myagent
	a, err := parseArgs([]string{"/p/x", "-G", "myagent"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--agent", "myagent"}
	if len(a.Passthrough) != len(want) || a.Passthrough[0] != want[0] || a.Passthrough[1] != want[1] {
		t.Errorf("Passthrough: got %v, want %v", a.Passthrough, want)
	}
}

func TestParseArgs_ShortRequired_Equals(t *testing.T) {
	// -G=myagent → --agent=myagent
	a, err := parseArgs([]string{"/p/x", "-G=myagent"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Passthrough) != 1 || a.Passthrough[0] != "--agent=myagent" {
		t.Errorf("Passthrough: got %v, want [--agent=myagent]", a.Passthrough)
	}
}

func TestParseArgs_ShortRequired_MissingValue_Error(t *testing.T) {
	_, err := parseArgs([]string{"/p/x", "-G"}, testHome)
	if err == nil {
		t.Fatal("expected error for -G with no value")
	}
}

func TestParseArgs_ShortRequired_NextIsFlag_Error(t *testing.T) {
	_, err := parseArgs([]string{"/p/x", "-G", "--something"}, testHome)
	if err == nil {
		t.Fatal("expected error for -G followed by flag")
	}
}

func TestParseArgs_ShortRequired_InMiddleOfCollapse_Error(t *testing.T) {
	// -GB is invalid: G is required-value but not last in group (B follows it)
	_, err := parseArgs([]string{"/p/x", "-GB", "val"}, testHome)
	if err == nil {
		t.Fatal("expected error for -GB (G not last in collapsed group)")
	}
}

func TestParseArgs_ShortOptional_NoValue(t *testing.T) {
	// -T with no following non-flag token → --tmux (no value)
	a, err := parseArgs([]string{"/p/x", "-T", "--verbose"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	// --tmux should be present; --verbose also in passthrough
	if len(a.Passthrough) < 1 || a.Passthrough[0] != "--tmux" {
		t.Errorf("Passthrough: got %v, expected --tmux first", a.Passthrough)
	}
}

func TestParseArgs_ShortOptional_WithValue_Space(t *testing.T) {
	// -T mywin → --tmux mywin (greedy)
	a, err := parseArgs([]string{"/p/x", "-T", "mywin"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--tmux", "mywin"}
	if len(a.Passthrough) != len(want) || a.Passthrough[0] != want[0] || a.Passthrough[1] != want[1] {
		t.Errorf("Passthrough: got %v, want %v", a.Passthrough, want)
	}
}

func TestParseArgs_ShortOptional_WithValue_Equals(t *testing.T) {
	// -T=mywin → --tmux=mywin
	a, err := parseArgs([]string{"/p/x", "-T=mywin"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Passthrough) != 1 || a.Passthrough[0] != "--tmux=mywin" {
		t.Errorf("Passthrough: got %v, want [--tmux=mywin]", a.Passthrough)
	}
}

func TestParseArgs_ShortOptional_AtEOF_NoValue(t *testing.T) {
	// -T at end of args → --tmux (no value)
	a, err := parseArgs([]string{"/p/x", "-T"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Passthrough) != 1 || a.Passthrough[0] != "--tmux" {
		t.Errorf("Passthrough: got %v, want [--tmux]", a.Passthrough)
	}
}

func TestParseArgs_ShortAllowedTools(t *testing.T) {
	// -W "Bash,Read" → --allowedTools "Bash,Read"
	a, err := parseArgs([]string{"/p/x", "-W", "Bash,Read"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--allowedTools", "Bash,Read"}
	if len(a.Passthrough) != len(want) || a.Passthrough[0] != want[0] || a.Passthrough[1] != want[1] {
		t.Errorf("Passthrough: got %v, want %v", a.Passthrough, want)
	}
}

func TestParseArgs_ShortPermissionMode(t *testing.T) {
	// -M=bypass-permissions → --permission-mode=bypass-permissions
	a, err := parseArgs([]string{"/p/x", "-M=bypass-permissions"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Passthrough) != 1 || a.Passthrough[0] != "--permission-mode=bypass-permissions" {
		t.Errorf("Passthrough: got %v", a.Passthrough)
	}
}

// ── Eaten-flag tests (--no-tmux / --no-permissions) ───────────────────────

func TestParseArgs_NoTmux_Eaten(t *testing.T) {
	a, err := parseArgs([]string{"/p/x", "--no-tmux"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if !a.NoTmux {
		t.Error("NoTmux: got false, want true")
	}
	// --no-tmux must not appear in passthrough
	assertNotContains(t, a.Passthrough, "--no-tmux")
}

func TestParseArgs_NoPermissions_Eaten(t *testing.T) {
	a, err := parseArgs([]string{"/p/x", "--no-permissions"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if !a.NoPermissions {
		t.Error("NoPermissions: got false, want true")
	}
	assertNotContains(t, a.Passthrough, "--no-permissions")
}

func TestParseArgs_NoTmuxAndNoPermissions_BothEaten(t *testing.T) {
	a, err := parseArgs([]string{"/p/x", "--no-tmux", "--no-permissions"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if !a.NoTmux {
		t.Error("NoTmux: got false, want true")
	}
	if !a.NoPermissions {
		t.Error("NoPermissions: got false, want true")
	}
	if len(a.Passthrough) != 0 {
		t.Errorf("Passthrough: got %v, want empty (eaten flags not passed through)", a.Passthrough)
	}
}

func TestParseArgs_NoTmux_DoesNotAffectExplicitT(t *testing.T) {
	// -T is still translated to --tmux even when --no-tmux is set
	a, err := parseArgs([]string{"/p/x", "--no-tmux", "-T"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if !a.NoTmux {
		t.Error("NoTmux: got false, want true")
	}
	assertContains(t, a.Passthrough, "--tmux")
}

// ── Auto-name wiring tests ─────────────────────────────────────────────────
// These tests exercise the shouldAutoName + heuristic path end-to-end by
// simulating what run() does: check shouldAutoName, prepend --name, then call
// buildArgv.  The LLM is not called (no ANTHROPIC_API_KEY set in tests).

func applyAutoName(a *Args, passthrough []string) {
	a.Passthrough = passthrough
	if shouldAutoName(a.Passthrough) {
		prompt := extractPrompt(a.Passthrough)
		// Use empty apiKey so we get the heuristic path (no real API call).
		// Stderr goes to a discard file; we don't test the warning here.
		name := heuristicName(prompt)
		a.Passthrough = append([]string{"--name", name}, a.Passthrough...)
	}
}

func TestAutoNameWiring_InjectsName(t *testing.T) {
	a := Args{CWD: "/p/main"}
	applyAutoName(&a, []string{"--", "fix the login bug"})

	argv := buildArgv(a, "/shell", defaultConfig())
	// --name should appear in argv.
	assertContains(t, argv, "--name")
	assertContains(t, argv, "fix-login-bug")
}

func TestAutoNameWiring_DoesNotInjectWhenNamePresent(t *testing.T) {
	a := Args{CWD: "/p/main"}
	applyAutoName(&a, []string{"--name", "my-session", "--", "fix bug"})

	argv := buildArgv(a, "/shell", defaultConfig())
	// Count --name occurrences — should be exactly 1 (the user-supplied one).
	count := 0
	for _, tok := range argv {
		if tok == "--name" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("--name appears %d times, want 1: %v", count, argv)
	}
}

func TestAutoNameWiring_DoesNotInjectWithResume(t *testing.T) {
	a := Args{CWD: "/p/main"}
	applyAutoName(&a, []string{"--resume", "abc123", "--", "fix bug"})

	argv := buildArgv(a, "/shell", defaultConfig())
	assertNotContains(t, argv, "--name")
}

func TestAutoNameWiring_DoesNotInjectWithPrint(t *testing.T) {
	a := Args{CWD: "/p/main"}
	applyAutoName(&a, []string{"--print", "--", "fix bug"})

	argv := buildArgv(a, "/shell", defaultConfig())
	assertNotContains(t, argv, "--name")
}

func TestAutoNameWiring_DoesNotInjectWithoutDashDash(t *testing.T) {
	a := Args{CWD: "/p/main"}
	applyAutoName(&a, []string{"--verbose"})

	argv := buildArgv(a, "/shell", defaultConfig())
	assertNotContains(t, argv, "--name")
}

func TestAutoNameWiring_NamePrependedBeforeOtherFlags(t *testing.T) {
	// --name <val> should appear before the user's other passthrough flags.
	a := Args{CWD: "/p/main"}
	applyAutoName(&a, []string{"--verbose", "--", "add dark mode"})

	// Find --name position.
	nameIdx := -1
	for i, tok := range a.Passthrough {
		if tok == "--name" {
			nameIdx = i
			break
		}
	}
	if nameIdx != 0 {
		t.Errorf("--name not at start of Passthrough; got %v", a.Passthrough)
	}
}

// ── helpers ────────────────────────────────────────────────────────────────

func assertArgv(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("argv len: got %d, want %d\n  got:  %v\n  want: %v", len(got), len(want), got, want)
		return
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("argv[%d]: got %q, want %q\n  full got:  %v\n  full want: %v", i, got[i], want[i], got, want)
		}
	}
}

func assertContains(t *testing.T, argv []string, token string) {
	t.Helper()
	for _, a := range argv {
		if a == token {
			return
		}
	}
	t.Errorf("argv missing %q: %v", token, argv)
}

func assertNotContains(t *testing.T, argv []string, token string) {
	t.Helper()
	for _, a := range argv {
		if a == token {
			t.Errorf("argv contains unexpected %q: %v", token, argv)
			return
		}
	}
}

// ── Parser tests: -w / --worktree ─────────────────────────────────────────

func TestParseArgs_W_Bare(t *testing.T) {
	// -w with no following non-flag token → WorktreeSet=true, WorktreeArg=""
	a, err := parseArgs([]string{"/p/x", "-w"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if !a.WorktreeSet {
		t.Error("WorktreeSet: got false, want true")
	}
	if a.WorktreeArg != "" {
		t.Errorf("WorktreeArg: got %q, want empty", a.WorktreeArg)
	}
	// -w must NOT appear in Passthrough (intercept holds it)
	assertNotContains(t, a.Passthrough, "-w")
	assertNotContains(t, a.Passthrough, "--worktree")
}

func TestParseArgs_W_WithValue_Space(t *testing.T) {
	// -w feat → WorktreeSet=true, WorktreeArg="feat"
	a, err := parseArgs([]string{"/p/x", "-w", "feat"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if !a.WorktreeSet {
		t.Error("WorktreeSet: got false, want true")
	}
	if a.WorktreeArg != "feat" {
		t.Errorf("WorktreeArg: got %q, want %q", a.WorktreeArg, "feat")
	}
	assertNotContains(t, a.Passthrough, "-w")
	assertNotContains(t, a.Passthrough, "--worktree")
}

func TestParseArgs_W_Equals(t *testing.T) {
	// -w=feat
	a, err := parseArgs([]string{"/p/x", "-w=feat"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if !a.WorktreeSet {
		t.Error("WorktreeSet: got false, want true")
	}
	if a.WorktreeArg != "feat" {
		t.Errorf("WorktreeArg: got %q, want %q", a.WorktreeArg, "feat")
	}
}

func TestParseArgs_Worktree_Bare(t *testing.T) {
	// --worktree with no value
	a, err := parseArgs([]string{"/p/x", "--worktree"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if !a.WorktreeSet {
		t.Error("WorktreeSet: got false, want true")
	}
	if a.WorktreeArg != "" {
		t.Errorf("WorktreeArg: got %q, want empty", a.WorktreeArg)
	}
}

func TestParseArgs_Worktree_WithValue_Space(t *testing.T) {
	// --worktree feat
	a, err := parseArgs([]string{"/p/x", "--worktree", "feat"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if !a.WorktreeSet {
		t.Error("WorktreeSet: got false, want true")
	}
	if a.WorktreeArg != "feat" {
		t.Errorf("WorktreeArg: got %q, want %q", a.WorktreeArg, "feat")
	}
}

func TestParseArgs_Worktree_Equals(t *testing.T) {
	// --worktree=feat
	a, err := parseArgs([]string{"/p/x", "--worktree=feat"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if !a.WorktreeSet {
		t.Error("WorktreeSet: got false, want true")
	}
	if a.WorktreeArg != "feat" {
		t.Errorf("WorktreeArg: got %q, want %q", a.WorktreeArg, "feat")
	}
}

func TestParseArgs_W_NextIsFlag_NoConsume(t *testing.T) {
	// -w followed by a flag token: value should not be consumed.
	a, err := parseArgs([]string{"/p/x", "-w", "--verbose"}, testHome)
	if err != nil {
		t.Fatal(err)
	}
	if !a.WorktreeSet {
		t.Error("WorktreeSet: got false, want true")
	}
	if a.WorktreeArg != "" {
		t.Errorf("WorktreeArg: got %q, want empty (flag not consumed)", a.WorktreeArg)
	}
	// --verbose should still be in passthrough
	assertContains(t, a.Passthrough, "--verbose")
}

// ── applyWorktreeIntercept tests ──────────────────────────────────────────

// fakeGitRunner returns a gitRunner replacement that serves a fixed worktree
// list output. Pass "" for out to simulate a git error.
func fakeGitRunner(out string) func(string, ...string) ([]byte, error) {
	return func(dir string, args ...string) ([]byte, error) {
		if out == "" {
			return nil, fmt.Errorf("not a git repo")
		}
		return []byte(out), nil
	}
}

// worktreeListOutput builds a fake `git worktree list --porcelain` output for
// a slice of absolute paths.
func worktreeListOutput(paths []string) string {
	var sb strings.Builder
	for i, p := range paths {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString("worktree ")
		sb.WriteString(p)
		sb.WriteString("\nHEAD abc123\nbranch refs/heads/main")
	}
	sb.WriteString("\n")
	return sb.String()
}

func TestApplyWorktreeIntercept_NotSet(t *testing.T) {
	orig := gitRunner
	defer func() { gitRunner = orig }()
	gitRunner = fakeGitRunner("") // should never be called

	a := Args{CWD: "/p/main"}
	applyWorktreeIntercept(&a, "/shell")
	if a.CWD != "/p/main" {
		t.Errorf("CWD changed unexpectedly: %q", a.CWD)
	}
	if len(a.Passthrough) != 0 {
		t.Errorf("Passthrough changed unexpectedly: %v", a.Passthrough)
	}
}

func TestApplyWorktreeIntercept_BareW_PushesThrough(t *testing.T) {
	orig := gitRunner
	defer func() { gitRunner = orig }()
	gitRunner = fakeGitRunner("") // bare -w: git never queried

	a := Args{CWD: "/p/main", WorktreeSet: true, WorktreeArg: ""}
	applyWorktreeIntercept(&a, "/shell")
	if a.WorktreeMatched {
		t.Error("WorktreeMatched should be false for bare -w")
	}
	assertContains(t, a.Passthrough, "--worktree")
}

func TestApplyWorktreeIntercept_MatchedExisting_SwapsCWD(t *testing.T) {
	orig := gitRunner
	defer func() { gitRunner = orig }()
	gitRunner = fakeGitRunner(worktreeListOutput([]string{"/repo/main", "/repo/feat"}))

	a := Args{CWD: "/repo/main", WorktreeSet: true, WorktreeArg: "feat"}
	applyWorktreeIntercept(&a, "/shell")

	if a.CWD != "/repo/feat" {
		t.Errorf("CWD: got %q, want /repo/feat", a.CWD)
	}
	if !a.WorktreeMatched {
		t.Error("WorktreeMatched: got false, want true")
	}
	assertNotContains(t, a.Passthrough, "--worktree")
	assertNotContains(t, a.Passthrough, "-w")
}

func TestApplyWorktreeIntercept_Unmatched_PassthroughPlusName(t *testing.T) {
	orig := gitRunner
	defer func() { gitRunner = orig }()
	gitRunner = fakeGitRunner(worktreeListOutput([]string{"/repo/main"}))

	a := Args{CWD: "/repo/main", WorktreeSet: true, WorktreeArg: "newfeature"}
	applyWorktreeIntercept(&a, "/shell")

	if a.WorktreeMatched {
		t.Error("WorktreeMatched: got true, want false")
	}
	assertContains(t, a.Passthrough, "--worktree")
	assertContains(t, a.Passthrough, "newfeature")
	assertContains(t, a.Passthrough, "--name")
}

func TestApplyWorktreeIntercept_Unmatched_NameAlreadySet_NoExtraName(t *testing.T) {
	orig := gitRunner
	defer func() { gitRunner = orig }()
	gitRunner = fakeGitRunner(worktreeListOutput([]string{"/repo/main"}))

	a := Args{
		CWD:         "/repo/main",
		WorktreeSet: true,
		WorktreeArg: "newfeature",
		Passthrough: []string{"--name", "myname"},
	}
	applyWorktreeIntercept(&a, "/shell")

	// --name already present; should not add another --name
	count := 0
	for _, t2 := range a.Passthrough {
		if t2 == "--name" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("--name appears %d times, want exactly 1: %v", count, a.Passthrough)
	}
}

func TestApplyWorktreeIntercept_NotARepo_PassthroughPlusName(t *testing.T) {
	orig := gitRunner
	defer func() { gitRunner = orig }()
	gitRunner = fakeGitRunner("") // simulate git error / not-a-repo

	a := Args{CWD: "/p/main", WorktreeSet: true, WorktreeArg: "newfeature"}
	applyWorktreeIntercept(&a, "/shell")

	if a.WorktreeMatched {
		t.Error("WorktreeMatched: got true, want false")
	}
	assertContains(t, a.Passthrough, "--worktree")
	assertContains(t, a.Passthrough, "--name")
}

// ── Agent-pitfall warning tests ────────────────────────────────────────────

func TestShouldAppendPitfallWarning_NoPrintFlag(t *testing.T) {
	if !shouldAppendPitfallWarning([]string{"--verbose", "--model", "sonnet"}) {
		t.Error("expected true for interactive session")
	}
}

func TestShouldAppendPitfallWarning_PrintShort(t *testing.T) {
	if shouldAppendPitfallWarning([]string{"-p", "do thing"}) {
		t.Error("expected false when -p present")
	}
}

func TestShouldAppendPitfallWarning_PrintLong(t *testing.T) {
	if shouldAppendPitfallWarning([]string{"--print", "do thing"}) {
		t.Error("expected false when --print present")
	}
}

func TestShouldAppendPitfallWarning_EmptyPassthrough(t *testing.T) {
	if !shouldAppendPitfallWarning([]string{}) {
		t.Error("expected true for empty passthrough")
	}
}

func TestWithAgentPitfallWarning_NoExisting_Appends(t *testing.T) {
	in := []string{"--verbose"}
	out := withAgentPitfallWarning(in)
	// Should be: [--verbose, --append-system-prompt, <warning>]
	if len(out) != 3 {
		t.Fatalf("len: got %d, want 3: %v", len(out), out)
	}
	if out[0] != "--verbose" {
		t.Errorf("[0]: got %q, want --verbose", out[0])
	}
	if out[1] != "--append-system-prompt" {
		t.Errorf("[1]: got %q, want --append-system-prompt", out[1])
	}
	if out[2] != agentPitfallWarning {
		t.Errorf("[2]: got %q, want warning", out[2])
	}
}

func TestWithAgentPitfallWarning_PrintMode_SkipsEntirely(t *testing.T) {
	in := []string{"-p", "prompt text"}
	out := withAgentPitfallWarning(in)
	// Should be unchanged.
	if len(out) != 2 || out[0] != "-p" || out[1] != "prompt text" {
		t.Errorf("expected unchanged passthrough, got %v", out)
	}
}

func TestWithAgentPitfallWarning_ExistingSpaceForm_Merges(t *testing.T) {
	in := []string{"--append-system-prompt", "user's text"}
	out := withAgentPitfallWarning(in)
	if len(out) != 2 {
		t.Fatalf("len: got %d, want 2: %v", len(out), out)
	}
	want := "user's text\n\n" + agentPitfallWarning
	if out[1] != want {
		t.Errorf("merged value: got %q, want %q", out[1], want)
	}
}

func TestWithAgentPitfallWarning_ExistingEqualsForm_Merges(t *testing.T) {
	in := []string{"--append-system-prompt=user's text"}
	out := withAgentPitfallWarning(in)
	if len(out) != 1 {
		t.Fatalf("len: got %d, want 1: %v", len(out), out)
	}
	want := "--append-system-prompt=user's text\n\n" + agentPitfallWarning
	if out[0] != want {
		t.Errorf("merged token: got %q, want %q", out[0], want)
	}
}

func TestWithAgentPitfallWarning_DoesNotMutateInput(t *testing.T) {
	in := []string{"--verbose", "--model", "sonnet"}
	_ = withAgentPitfallWarning(in)
	if len(in) != 3 || in[0] != "--verbose" {
		t.Errorf("input mutated: %v", in)
	}
}

func TestBuildArgv_IncludesPitfallWarning_DefaultConfig(t *testing.T) {
	a := Args{CWD: "/p/main"}
	argv := buildArgv(a, "/shell", Config{})
	assertContains(t, argv, "--append-system-prompt")
	assertContains(t, argv, agentPitfallWarning)
}

func TestBuildArgv_OmitsPitfallWarning_WhenPrintMode(t *testing.T) {
	a := Args{
		CWD:         "/p/main",
		Passthrough: []string{"-p", "the prompt"},
	}
	argv := buildArgv(a, "/shell", Config{})
	assertNotContains(t, argv, "--append-system-prompt")
}
