package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Args holds the result of parsing fnclaude's own argv.
type Args struct {
	// CWD is the directory claude will be launched in (first positional, or
	// ~/.claude/noop when no positionals given).
	CWD string

	// ExtraDirs collects positional[1:] and all -A/--also values, in order.
	ExtraDirs []string

	// Passthrough is everything else, preserved in order, to be forwarded to
	// claude verbatim. Short flags are already translated to their long forms.
	Passthrough []string

	// NoTmux is true when the user passed --no-tmux (eaten by fnclaude; not
	// forwarded to claude).
	NoTmux bool

	// NoPermissions is true when the user passed --no-permissions (eaten by
	// fnclaude; not forwarded to claude).
	NoPermissions bool

	// WorktreeSet is true when the user passed -w / --worktree.
	// The intercept logic (applyWorktreeIntercept) decides what to do with it.
	WorktreeSet bool

	// WorktreeArg is the name/value given with -w / --worktree, or "" if the
	// flag was bare (no value provided).
	WorktreeArg string

	// WorktreeMatched is set to true by applyWorktreeIntercept when the worktree
	// name matched an existing worktree and the CWD was swapped. False means the
	// flag was passed through (new worktree being created, or bare -w).
	WorktreeMatched bool
}

// modelAliases is the set of magic tokens that map to --model.
var modelAliases = map[string]bool{
	"opus":   true,
	"sonnet": true,
	"haiku":  true,
}

// effortLevels is the set of magic tokens that map to --effort.
var effortLevels = map[string]bool{
	"low":    true,
	"medium": true,
	"high":   true,
	"xhigh":  true,
	"max":    true,
}

// shortNoValue maps capital short flags (no value) to their long forms.
var shortNoValue = map[byte]string{
	'B': "--brief",
	'C': "--chrome",
	'D': "--dangerously-skip-permissions",
	'F': "--fork-session",
	'I': "--ide",
	'V': "--verbose",
}

// shortRequired maps capital short flags (required value) to their long forms.
var shortRequired = map[byte]string{
	'G': "--agent",
	'M': "--permission-mode",
	'W': "--allowedTools",
}

// shortOptional maps capital short flags (optional value) to their long forms.
var shortOptional = map[byte]string{
	'P': "--from-pr",
	'R': "--remote-control",
	'T': "--tmux",
}

// subcommandFlags maps subcommand-style positional tokens to the long flag
// they expand to. Recognized only in positional territory (before the first
// flag-shaped token); shadowed by a leading "./" prefix for literal-path
// disambiguation, same as the model/effort magic.
var subcommandFlags = map[string]string{
	"resume":   "--resume",
	"res":      "--resume",
	"continue": "--continue",
	"con":      "--continue",
}

// parseArgs parses os.Args[1:] and returns the structured result.
// home is the value to use for the noop fallback (typically os.UserHomeDir()).
//
// Magic positional rules (strictly positional, not last-wins):
//   - Position 1: if it exactly matches a model alias → --model <alias>, continue
//     to position 2. Otherwise → position 1 is the cwd, magic off.
//   - Position 2 (only when position 1 was a model alias): if it exactly matches
//     an effort level → --effort <level>, magic off. Otherwise → position 2 is
//     the cwd, magic off.
//   - Position 3+: never magic; normal positional parsing (extra dirs).
//
// Effort without a preceding model alias is NOT magic — it becomes the cwd.
//
// Subcommand-style positionals (`resume`, `res`, `continue`, `con`) may appear
// at ANY positional slot, independent of magic state. They consume the slot
// without advancing magic, so `fnc resume opus xhigh` and `fnc opus xhigh resume`
// parse equivalently. At most one subcommand per invocation; a second is an error.
func parseArgs(argv []string, home string) (Args, error) {
	var firstPath string
	var extraDirs []string
	var passthrough []string
	var noTmux bool
	var noPermissions bool
	var worktreeSet bool
	var worktreeArg string

	// Magic slots: filled at most once each, in strict order.
	magicModel := ""
	magicEffort := ""
	// subcommandFlag is the long flag a subcommand-style positional expanded
	// to (e.g. `resume` → `--resume`). Set at most once; a second hit errors.
	subcommandFlag := ""
	subcommandToken := ""

	// magicState tracks where we are in the magic scanning sequence.
	//   0 = position 1 (check model)
	//   1 = position 2 (check effort, only if model matched)
	//   2 = magic done
	magicState := 0

	inFlags := false // once true, non-flag tokens go to passthrough
	firstPathSet := false

	i := 0
	for i < len(argv) {
		arg := argv[i]

		// ── Positional collection (before first flag-shaped token) ───────────
		if !inFlags && !strings.HasPrefix(arg, "-") {
			// Subcommand check fires at every positional slot, independent of
			// magic state. Doesn't advance magicState — `fnc resume opus xhigh`
			// and `fnc opus xhigh resume` parse equivalently.
			if flag, ok := subcommandFlags[arg]; ok {
				if subcommandFlag != "" {
					return Args{}, fmt.Errorf(
						"fnclaude: only one subcommand allowed (got %q and %q)",
						subcommandToken, arg)
				}
				subcommandFlag = flag
				subcommandToken = arg
				i++
				continue
			}
			// Magic scanning at position 1: model alias check.
			if magicState == 0 {
				if modelAliases[arg] {
					magicModel = arg
					magicState = 1 // advance to effort check at position 2
					i++
					continue
				}
				// Not a model alias — position 1 is the cwd; magic done.
				magicState = 2
			} else if magicState == 1 {
				// Magic scanning at position 2: effort level check (model matched).
				if effortLevels[arg] {
					magicEffort = arg
					magicState = 2 // magic done after position 2
					i++
					continue
				}
				// Not an effort level — position 2 is the cwd; magic done.
				magicState = 2
			}
			// magicState == 2: normal positional.

			if !firstPathSet {
				firstPath = arg
				firstPathSet = true
			} else {
				extraDirs = append(extraDirs, arg)
			}
			i++
			continue
		}

		// Anything from here on: we are in flag territory.
		inFlags = true

		// ── fnclaude-eaten flags (not forwarded to claude) ───────────────────
		if arg == "--no-tmux" {
			noTmux = true
			i++
			continue
		}
		if arg == "--no-permissions" {
			noPermissions = true
			i++
			continue
		}

		// ── -A / --also ──────────────────────────────────────────────────────
		// Supported forms: -A <val>, -A=<val>, --also <val>, --also=<val>
		if arg == "-A" || arg == "--also" {
			if i+1 >= len(argv) || strings.HasPrefix(argv[i+1], "-") {
				which := arg
				if i+1 < len(argv) {
					which = fmt.Sprintf("%s %s", arg, argv[i+1])
				}
				return Args{}, fmt.Errorf("fnclaude: %s requires a directory argument", which)
			}
			extraDirs = append(extraDirs, argv[i+1])
			i += 2
			continue
		}
		if strings.HasPrefix(arg, "-A=") {
			val := arg[len("-A="):]
			if val == "" {
				return Args{}, fmt.Errorf("fnclaude: -A= requires a directory argument")
			}
			extraDirs = append(extraDirs, val)
			i++
			continue
		}
		if strings.HasPrefix(arg, "--also=") {
			val := arg[len("--also="):]
			if val == "" {
				return Args{}, fmt.Errorf("fnclaude: --also= requires a directory argument")
			}
			extraDirs = append(extraDirs, val)
			i++
			continue
		}

		// ── -w / --worktree ──────────────────────────────────────────────────
		// Intercepted by fnclaude; NOT pushed to passthrough here.
		// Supported forms:
		//   -w            (bare, no value)
		//   -w <val>      (space-separated value)
		//   -w=<val>      (equals form)
		//   --worktree    (bare, no value)
		//   --worktree <val>
		//   --worktree=<val>
		if arg == "-w" || arg == "--worktree" {
			worktreeSet = true
			// Greedy: if next token is non-flag and non-empty, consume as value.
			if i+1 < len(argv) && !strings.HasPrefix(argv[i+1], "-") {
				worktreeArg = argv[i+1]
				i += 2
			} else {
				i++
			}
			continue
		}
		if strings.HasPrefix(arg, "-w=") {
			worktreeSet = true
			worktreeArg = arg[len("-w="):]
			i++
			continue
		}
		if strings.HasPrefix(arg, "--worktree=") {
			worktreeSet = true
			worktreeArg = arg[len("--worktree="):]
			i++
			continue
		}

		// ── Single-dash short flags ───────────────────────────────────────────
		if len(arg) >= 2 && arg[0] == '-' && arg[1] != '-' {
			tokens, n, err := parseShortFlag(arg, argv[i+1:])
			if err != nil {
				return Args{}, err
			}
			passthrough = append(passthrough, tokens...)
			i += 1 + n
			continue
		}

		// ── Everything else: passthrough ─────────────────────────────────────
		passthrough = append(passthrough, arg)
		i++
	}

	// Prepend --model / --effort tokens from magic positionals (last-wins
	// means the magic vars already hold the correct final value).
	var magicPrefix []string
	if magicModel != "" {
		magicPrefix = append(magicPrefix, "--model", magicModel)
	}
	if magicEffort != "" {
		magicPrefix = append(magicPrefix, "--effort", magicEffort)
	}
	if len(magicPrefix) > 0 {
		passthrough = append(magicPrefix, passthrough...)
	}

	// Prepend the subcommand flag (--resume / --continue). Lands at index 0
	// so it stays in front of any `--` separator a prompt brought in.
	if subcommandFlag != "" {
		passthrough = append([]string{subcommandFlag}, passthrough...)
	}

	// CWD fallback.
	cwd := filepath.Join(home, ".claude", "noop")
	if firstPathSet {
		cwd = firstPath
	}

	return Args{
		CWD:           cwd,
		ExtraDirs:     extraDirs,
		Passthrough:   passthrough,
		NoTmux:        noTmux,
		NoPermissions: noPermissions,
		WorktreeSet:   worktreeSet,
		WorktreeArg:   worktreeArg,
	}, nil
}

// parseShortFlag parses a short-flag token (e.g. "-B", "-BV", "-G=val",
// "-G", "-Gval") and returns the passthrough tokens to emit plus the number
// of additional argv elements consumed from rest (the slice starting after
// the current token).
func parseShortFlag(arg string, rest []string) ([]string, int, error) {
	// Strip leading '-'.
	body := arg[1:]

	// Handle -X=val form (equals in the token itself). Only valid for
	// single-char flags that take a value.
	if eqIdx := strings.IndexByte(body, '='); eqIdx == 1 {
		ch := body[0]
		val := body[eqIdx+1:]
		long, ok := valueShortLong(ch)
		if !ok {
			// Not a known value-taking flag; fall through to general handling.
			// (This shouldn't happen in practice for our flag set, but be safe.)
			return []string{arg}, 0, nil
		}
		// Emit --long=val (single token, preserving = form).
		return []string{long + "=" + val}, 0, nil
	}

	// General case: body is one or more chars.
	// Walk each char; the last one may take a value from rest.
	var out []string
	for pos := 0; pos < len(body); pos++ {
		ch := body[pos]
		isLast := pos == len(body)-1

		if long, ok := shortNoValue[ch]; ok {
			out = append(out, long)
			continue
		}

		if long, ok := shortRequired[ch]; ok {
			if !isLast {
				return nil, 0, fmt.Errorf("fnclaude: flag -%c cannot be in middle of collapsed group, requires a value", ch)
			}
			// Consume next token as value.
			if len(rest) == 0 || strings.HasPrefix(rest[0], "-") {
				return nil, 0, fmt.Errorf("fnclaude: -%c requires a value", ch)
			}
			// Emit --long val (two tokens, space form).
			return append(out, long, rest[0]), 1, nil
		}

		if long, ok := shortOptional[ch]; ok {
			if !isLast {
				return nil, 0, fmt.Errorf("fnclaude: flag -%c cannot be in middle of collapsed group, requires a value", ch)
			}
			// Greedy: if next token doesn't start with '-', consume it.
			if len(rest) > 0 && !strings.HasPrefix(rest[0], "-") {
				// Emit --long val (two tokens, space form).
				return append(out, long, rest[0]), 1, nil
			}
			// No value — emit just --long.
			return append(out, long), 0, nil
		}

		// Unknown short flag; pass through verbatim (let claude handle or error).
		out = append(out, "-"+string(ch))
	}

	return out, 0, nil
}

// valueShortLong returns the long form for a value-taking short flag, and
// whether it was found.
func valueShortLong(ch byte) (string, bool) {
	if long, ok := shortRequired[ch]; ok {
		return long, true
	}
	if long, ok := shortOptional[ch]; ok {
		return long, true
	}
	return "", false
}

// settingSourcesInPassthrough returns true if any passthrough token is
// --setting-sources or starts with --setting-sources=.
func settingSourcesInPassthrough(passthrough []string) bool {
	for _, t := range passthrough {
		if t == "--setting-sources" || strings.HasPrefix(t, "--setting-sources=") {
			return true
		}
	}
	return false
}

// tokenInPassthrough returns true if the exact token appears in passthrough,
// or if any token has the form token=<anything>.
func tokenInPassthrough(passthrough []string, long string) bool {
	for _, t := range passthrough {
		if t == long || strings.HasPrefix(t, long+"=") {
			return true
		}
	}
	return false
}

// nameInPassthrough returns true if --name or -n (with or without =value)
// appears in the passthrough slice.
func nameInPassthrough(passthrough []string) bool {
	for _, t := range passthrough {
		if t == "--name" || t == "-n" ||
			strings.HasPrefix(t, "--name=") || strings.HasPrefix(t, "-n=") {
			return true
		}
	}
	return false
}

// version is the binary's version string. Default "dev" for local `go build`;
// goreleaser injects the release tag via -ldflags -X main.version=v0.X.Y.
var version = "dev"

// wantsVersion returns true when the user passed -v or --version anywhere
// in argv before a literal "--" terminator. fnclaude shadows claude's -v
// short flag (the only lowercase short fnclaude claims); to reach claude's
// own --version, the user runs `claude --version` directly.
func wantsVersion(argv []string) bool {
	for _, t := range argv {
		if t == "--" {
			return false
		}
		if t == "-v" || t == "--version" {
			return true
		}
	}
	return false
}

// helpText is what `fnclaude --help` / `fnclaude -h` prints.
const helpText = `fnclaude — claude CLI launcher with quality-of-life features

Usage:
  fnclaude [MODEL] [EFFORT] [CWD [EXTRA_DIRS...]] [FLAGS...] [-- PROMPT]

Magic positional words (positions 1+2 only, before any path):
  Position 1 — model alias: opus | sonnet | haiku            → --model <alias>
  Position 2 — effort level: low | medium | high | xhigh | max → --effort <level>
                              (only honored when position 1 was a model alias)
  To use a directory literally named opus/max/etc., prefix with ./

Subcommand positionals (any positional slot, max one per invocation):
  resume | res        → --resume       (claude shows the session picker)
  continue | con      → --continue     (resumes the most recent session)
  Order-independent: "fnc resume opus" and "fnc opus resume" parse equivalently.
  To use a directory literally named resume/continue/res/con, prefix with ./

Positional paths:
  First path  → cwd to launch claude in (fallback ~/.claude/noop)
  Extra paths → --add-dir; .mcp.json and .claude/settings.json auto-injected
                if those files exist in the extra-dir

fnclaude-owned flags:
  -A, --also <dir>     additional extra-dir (repeatable; same effect as 2nd+ positional)
      --no-tmux        suppress auto-tmux injection for this invocation
      --no-permissions suppress auto-DSP injection for this invocation
  -h, --help           show this help
  -v, --version        print fnclaude's version and exit
                       (shadows claude's -v; use ` + "`claude --version`" + ` directly for that)

Capital-letter shortcuts (translate to claude long-form flags):
  -B → --brief                          -M → --permission-mode <mode>
  -C → --chrome                         -P → --from-pr [value]
  -D → --dangerously-skip-permissions   -R → --remote-control [name]
  -F → --fork-session                   -T → --tmux [classic]
  -G → --agent <agent>                  -V → --verbose
  -I → --ide                            -W → --allowedTools <tools>

All other claude flags pass through verbatim — run ` + "`claude --help`" + ` for the full
reference. POSIX collapsing is supported (-BVC = -B -V -C); only the last flag in
a collapsed group may take a value.

Cross-cwd resume: when claude shows the resume picker and you select a session
from a different cwd, fnclaude transparently re-launches in that cwd.

Worktree intercept: -w <name> matching an existing worktree of the project repo
swaps fnclaude's cwd to that worktree. Non-matching names pass through and the
new worktree's name is also set as the session --name.

Auto-name: when --, a prompt, and no --name/-n flag are all present, fnclaude
generates a 1-3 word session label via Haiku (falling back to a heuristic).
Requires ANTHROPIC_API_KEY; warns if missing.

Config file:
  $XDG_CONFIG_HOME/fnclaude/config.toml (or ~/.config/fnclaude/config.toml)
  [exec.env] NAME = "value" entries are injected into claude's environment.

Environment variables (override config; precedence: CLI > env > config > default):
  ANTHROPIC_API_KEY                       auth for auto-name LLM call
  FNCLAUDE_NAME_MODEL                     model for auto-name (default: claude-haiku-4-5)
  FNCLAUDE_NAME_TIMEOUT                   auto-name LLM timeout (default: 3s)
  FNCLAUDE_QUIET_MISSING_API_KEY          silence the "API key not set" warning
  FNCLAUDE_TMUX                           never | worktree | always (default: never)
  FNCLAUDE_IDE                            never | always (default: never)
  FNCLAUDE_DANGEROUSLY_SKIP_PERMISSIONS   true to auto-add --dangerously-skip-permissions

Examples:
  fnclaude                                # interactive in ~/.claude/noop
  fnclaude opus max ~/src/proj            # opus + max effort, launch in ~/src/proj
  fnclaude ~/src/a ~/src/b                # main + extra dir with mcp/settings injection
  fnclaude ~/src/proj -- "fix the bug"    # auto-name from prompt
  fnclaude -A docs/ ~/src/proj -V         # ergonomic flag form

For more, see https://github.com/fnrhombus/fnclaude
`

// wantsHelp returns true when the user passed -h or --help anywhere in argv
// before a literal "--" terminator. Tokens after "--" are part of the prompt
// to claude and aren't fnclaude flags.
func wantsHelp(argv []string) bool {
	for _, t := range argv {
		if t == "--" {
			return false
		}
		if t == "-h" || t == "--help" {
			return true
		}
	}
	return false
}

// agentPitfallWarning is appended to claude's system prompt via
// --append-system-prompt for any interactive session. It addresses a real
// foot-gun in Claude Code's Agent tool: when isolation is set to "worktree",
// an agent whose prompt names a path will cd to that path and silently
// bypass its isolated working directory.
const agentPitfallWarning = `When using the Task / Agent tool with isolation set to "worktree", do not name the main repo's absolute path in the spawn prompt — the agent will cd there and silently bypass its isolated worktree. Phrase locations as "your worktree" or "this directory", and verify with git log on the worktree branch after the agent reports done.`

// shouldAppendPitfallWarning reports whether the agent-pitfall warning is
// relevant for this invocation. Skipped for -p / --print (one-shot
// non-interactive) where agent spawning is unusual and the system-prompt
// overhead isn't worth paying.
func shouldAppendPitfallWarning(passthrough []string) bool {
	for _, t := range passthrough {
		if t == "-p" || t == "--print" {
			return false
		}
	}
	return true
}

// withAgentPitfallWarning returns a copy of passthrough with the
// agent-pitfall warning injected into --append-system-prompt. If the user
// already passed --append-system-prompt, the warning is concatenated to
// their value (separated by a blank line). Skipped entirely when
// shouldAppendPitfallWarning is false.
func withAgentPitfallWarning(passthrough []string) []string {
	if !shouldAppendPitfallWarning(passthrough) {
		return passthrough
	}
	for i, t := range passthrough {
		if t == "--append-system-prompt" && i+1 < len(passthrough) {
			out := append([]string{}, passthrough...)
			out[i+1] = passthrough[i+1] + "\n\n" + agentPitfallWarning
			return out
		}
		if strings.HasPrefix(t, "--append-system-prompt=") {
			existing := t[len("--append-system-prompt="):]
			out := append([]string{}, passthrough...)
			out[i] = "--append-system-prompt=" + existing + "\n\n" + agentPitfallWarning
			return out
		}
	}
	return append(append([]string{}, passthrough...), "--append-system-prompt", agentPitfallWarning)
}

// gitRunner is a function type that runs a git command in a directory and
// returns stdout. It is a variable so tests can substitute a fake.
var gitRunner = func(dir string, args ...string) ([]byte, error) {
	cmdArgs := append([]string{"-C", dir}, args...)
	return exec.Command("git", cmdArgs...).Output()
}

// worktreeInfo is one entry from `git worktree list --porcelain`.
type worktreeInfo struct {
	Path   string // absolute filesystem path
	Branch string // bare branch name (e.g. "feature-x" or "worktree-feature-x"); "" if detached
}

// listWorktrees returns one worktreeInfo per worktree by running
// `git worktree list --porcelain` in dir. Returns nil on any error (not-a-repo,
// etc.) with no error propagated — callers treat nil as "no match possible".
func listWorktrees(dir string) []worktreeInfo {
	out, err := gitRunner(dir, "worktree", "list", "--porcelain")
	if err != nil {
		return nil
	}

	var result []worktreeInfo
	// Output is blank-line-separated blocks. Each block has lines like:
	//   worktree /absolute/path
	//   HEAD <sha>
	//   branch refs/heads/<branchname>     (or "detached" instead)
	for _, block := range strings.Split(string(out), "\n\n") {
		var wt worktreeInfo
		for _, line := range strings.Split(block, "\n") {
			switch {
			case strings.HasPrefix(line, "worktree "):
				wt.Path = strings.TrimSpace(strings.TrimPrefix(line, "worktree "))
			case strings.HasPrefix(line, "branch refs/heads/"):
				wt.Branch = strings.TrimSpace(strings.TrimPrefix(line, "branch refs/heads/"))
			}
		}
		if wt.Path != "" {
			result = append(result, wt)
		}
	}
	return result
}

// findWorktree picks the worktreeInfo matching query, trying three strategies
// in order. Branch name is checked first since the branch is the semantically
// stable identifier for a worktree — its path can be anywhere the creator
// chose, but its branch is the same string the user typed at creation time.
//
//  1. Branch name           ==  query  (matches any worktree, any convention)
//  2. Branch with `worktree-` prefix stripped == query  (matches Claude's
//     default `worktree-<name>` branches)
//  3. Basename of the path  ==  query  (last-resort fallback for worktrees
//     whose branch was renamed or whose
//     creator skipped the convention)
//
// Returns nil when no entry matches.
func findWorktree(worktrees []worktreeInfo, query string) *worktreeInfo {
	if query == "" {
		// Defensive: never match against detached worktrees (Branch="") or
		// empty basenames just because the caller passed an empty query.
		// applyWorktreeIntercept already short-circuits on empty WorktreeArg
		// upstream of this; this guard keeps the helper safe in isolation.
		return nil
	}
	for i := range worktrees {
		if worktrees[i].Branch == query {
			return &worktrees[i]
		}
	}
	for i := range worktrees {
		if worktrees[i].Branch != "" && strings.TrimPrefix(worktrees[i].Branch, "worktree-") == query {
			return &worktrees[i]
		}
	}
	for i := range worktrees {
		if filepath.Base(worktrees[i].Path) == query {
			return &worktrees[i]
		}
	}
	return nil
}

// applyWorktreeIntercept applies the -w/--worktree intercept logic to a.
// It may modify a.CWD and a.Passthrough in place, and sets
// a.WorktreeMatched=true when an existing worktree was matched (cwd swapped).
//
// shellCWD is the process working directory (os.Getwd()) used to resolve a
// relative a.CWD to an absolute path before querying git.
func applyWorktreeIntercept(a *Args, shellCWD string) {
	if !a.WorktreeSet {
		return
	}

	// Bare -w with no name: push --worktree back through unchanged.
	if a.WorktreeArg == "" {
		a.Passthrough = append(a.Passthrough, "--worktree")
		return
	}

	// Resolve absolute cwd for git queries.
	dir := a.CWD
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(shellCWD, dir)
	}

	// List worktrees in the project repo, then match the user's query against
	// basename / branch / worktree-stripped-branch.
	if hit := findWorktree(listWorktrees(dir), a.WorktreeArg); hit != nil {
		// Existing worktree matched: swap cwd, suppress -w.
		a.CWD = hit.Path
		a.WorktreeMatched = true
		return
	}

	// No match (or not a repo): pass --worktree through and attach --name.
	a.Passthrough = append(a.Passthrough, "--worktree", a.WorktreeArg)
	if !nameInPassthrough(a.Passthrough) {
		a.Passthrough = append(a.Passthrough, "--name", a.WorktreeArg)
	}
}

// buildArgv constructs the argv slice to exec claude with, given the parsed
// args, the user's shell cwd (used to resolve relative extra-dir paths), and
// the loaded config.
// shellCWD is the process working directory at fnclaude startup (os.Getwd()).
func buildArgv(a Args, shellCWD string, cfg Config) []string {
	suppressSettings := settingSourcesInPassthrough(a.Passthrough)

	argv := []string{"claude"}

	// --dangerously-skip-permissions is added only when:
	//   - User explicitly passed -D (→ --dangerously-skip-permissions in passthrough), OR
	//   - auto.dangerously_skip_permissions is true AND --no-permissions not eaten.
	// User-explicit -D wins over --no-permissions (it's already in passthrough, stays there).
	// This block only handles the auto-injection case; the explicit case is in passthrough.
	if !tokenInPassthrough(a.Passthrough, "--dangerously-skip-permissions") &&
		cfg.Auto.DangerouslySkipPermissions && !a.NoPermissions {
		argv = append(argv, "--dangerously-skip-permissions")
	}

	// Inject --add-dir (and optional --mcp-config / --settings) for each
	// extra dir. Paths are resolved relative to the user's shell cwd.
	for _, d := range a.ExtraDirs {
		if !filepath.IsAbs(d) {
			d = filepath.Join(shellCWD, d)
		}
		argv = append(argv, "--add-dir", d)

		mcpConfig := filepath.Join(d, ".mcp.json")
		if _, err := os.Stat(mcpConfig); err == nil {
			argv = append(argv, "--mcp-config", mcpConfig)
		}

		if !suppressSettings {
			settings := filepath.Join(d, ".claude", "settings.json")
			if _, err := os.Stat(settings); err == nil {
				argv = append(argv, "--settings", settings)
			}
		}
	}

	// Auto-inject --ide if auto.ide == "always" and --ide not already in passthrough.
	if cfg.Auto.IDE == "always" && !tokenInPassthrough(a.Passthrough, "--ide") {
		argv = append(argv, "--ide")
	}

	// Auto-inject --tmux based on auto.tmux config.
	//
	// claude requires --worktree to be present when --tmux is used. The only
	// auto mode that's compatible with that constraint is "worktree", which
	// fires when the user is already creating a new worktree themselves:
	//
	//   "worktree" — inject --tmux when the user passed -w / --worktree for
	//                a NEW worktree (a.WorktreeSet && !a.WorktreeMatched).
	//                --worktree is already in passthrough; claude's constraint
	//                is satisfied without fnclaude generating worktrees on
	//                its own.
	//   "never"    — no-op.
	//
	// fnclaude never auto-creates worktrees — that's always a user-initiated
	// action.
	if cfg.Auto.Tmux == "worktree" &&
		!tokenInPassthrough(a.Passthrough, "--tmux") &&
		!a.NoTmux &&
		a.WorktreeSet && !a.WorktreeMatched {
		argv = append(argv, "--tmux")
	}

	argv = append(argv, withAgentPitfallWarning(a.Passthrough)...)

	return argv
}

func run() int {
	// Flush any deferred warnings on exit, AFTER claude has finished and the
	// user is back at their shell — never during startup where the warnings
	// would scroll off-screen behind claude's UI. The silent-relaunch path
	// (cross-cwd resume) uses syscall.Exec, which skips this defer; that's
	// intentional, since the relaunched fnclaude will re-emit any warnings
	// that still apply.
	defer flushWarnings()

	if wantsHelp(os.Args[1:]) {
		fmt.Print(helpText)
		return 0
	}

	if wantsVersion(os.Args[1:]) {
		fmt.Println("fnclaude " + version)
		return 0
	}

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fnclaude: cannot determine home directory: %v\n", err)
		return 1
	}

	a, err := parseArgs(os.Args[1:], home)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	cfg := loadConfig()

	shellCWD, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fnclaude: cannot determine working directory: %v\n", err)
		return 1
	}

	// Apply -w/--worktree intercept (may modify a.CWD and a.Passthrough).
	applyWorktreeIntercept(&a, shellCWD)

	// Resolve the launch cwd relative to the shell cwd.
	launchCWD := a.CWD
	if !filepath.IsAbs(launchCWD) {
		launchCWD = filepath.Join(shellCWD, launchCWD)
	}

	// Auto-name: if the invocation qualifies, generate a session name and
	// prepend --name <name> to the passthrough slice before buildArgv runs.
	if shouldAutoName(a.Passthrough) {
		prompt := extractPrompt(a.Passthrough)
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		name := generateName(prompt, cfg.Name, apiKey, nil)
		a.Passthrough = append([]string{"--name", name}, a.Passthrough...)
	}

	// Sanitize any --name/-n value (user-supplied or injected) to a
	// path-safe slug. Defense-in-depth before the value flows to claude
	// and potentially into the worktree-paths plugin's WorktreeCreate hook.
	var nameWarns []string
	a.Passthrough, nameWarns = sanitizeNamesInPassthrough(a.Passthrough)
	for _, w := range nameWarns {
		warn("%s", w)
	}

	argv := buildArgv(a, shellCWD, cfg)

	// Verify claude is on PATH before starting the PTY (gives a clean error).
	if _, err := exec.LookPath("claude"); err != nil {
		fmt.Fprintf(os.Stderr, "fnclaude: claude not found in PATH: %v\n", err)
		return 1
	}

	exitCode, tail := runWithPTY(argv, launchCWD, cfg)

	// Detect the cross-cwd redirect message that claude emits when the user
	// picks a session from a different directory via the Ctrl+A picker.
	if dest, uuid, ok := detectCrossCwd(tail); ok {
		// silentRelaunch replaces the process image via syscall.Exec and
		// never returns on success.  On failure it logs to stderr.
		silentRelaunch(os.Args[1:], dest, uuid)
		// If we get here silentRelaunch failed; propagate claude's exit code.
	}

	return exitCode
}

func main() {
	os.Exit(run())
}
