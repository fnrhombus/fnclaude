package main

import (
	"path/filepath"
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
	want := []string{"claude", "--dangerously-skip-permissions", "--verbose"}
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
	want := []string{"claude", "--dangerously-skip-permissions", "--add-dir", "/p/extra"}
	assertArgv(t, argv, want)
}

func TestBuildArgv_RelativeExtraDirResolved(t *testing.T) {
	a := Args{
		CWD:       "/p/main",
		ExtraDirs: []string{"relative/dir"},
	}
	argv := buildArgv(a, "/shell/cwd", defaultConfig())
	// The relative path should be joined with shellCWD.
	want := []string{"claude", "--dangerously-skip-permissions", "--add-dir", "/shell/cwd/relative/dir"}
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
		"claude", "--dangerously-skip-permissions",
		"--add-dir", "/p/b",
		"--add-dir", "/p/c",
	}
	assertArgv(t, argv, want)
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
