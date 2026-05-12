# Fish completion for fnclaude (and its fnc alias).
#
# Install: copy this file to ~/.config/fish/completions/fnclaude.fish
# For the fnc alias, also copy or symlink it as fnclaude.fish and fnc.fish,
# or use the separate alias registration at the bottom of this file.

# Disable file completion by default; we add it back selectively.
complete -c fnclaude -f
complete -c fnc     -f

# ---------------------------------------------------------------------------
# Helper: emit basenames of all git worktrees in the current repo.
# Outputs nothing (no error) when not in a git repo.
# ---------------------------------------------------------------------------

function __fnclaude_worktree_names
    set -l names
    for line in (git worktree list --porcelain 2>/dev/null)
        if string match -q 'worktree *' -- $line
            set -l path (string replace 'worktree ' '' -- $line)
            set names $names (basename $path)
        end
    end
    string join \n $names
end

# ---------------------------------------------------------------------------
# Flags — no value
# ---------------------------------------------------------------------------

complete -c fnclaude -n '__fish_use_subcommand' -l no-tmux          -d 'disable tmux integration'
complete -c fnclaude -n '__fish_use_subcommand' -l no-permissions    -d 'disable auto dangerously-skip-permissions injection'
complete -c fnclaude -s B -l brief              -d 'brief output mode'
complete -c fnclaude -s C -l chrome             -d 'open in Chrome'
complete -c fnclaude -s D -l dangerously-skip-permissions -d 'skip all permission prompts'
complete -c fnclaude -s F -l fork-session       -d 'fork the current session'
complete -c fnclaude -s I -l ide                -d 'enable IDE integration'
complete -c fnclaude -s V -l verbose            -d 'verbose output'

# ---------------------------------------------------------------------------
# Flags — required argument
# ---------------------------------------------------------------------------

complete -c fnclaude -s A -l also -r -a '(__fish_complete_directories)' -d 'add an extra directory'
complete -c fnclaude -s G -l agent -r          -d 'run a named agent'
complete -c fnclaude -s W -l allowedTools -r   -d 'restrict allowed tools'

# --permission-mode / -M enum
complete -c fnclaude -s M -l permission-mode -r -a 'acceptEdits\tauto-accept file edits
auto\tautomatic permission handling
bypassPermissions\tbypass all checks
default\tdefault handling
dontAsk\tnever ask
plan\tplan (read-only) mode' -d 'set permission mode'

# ---------------------------------------------------------------------------
# Flags — optional argument
# ---------------------------------------------------------------------------

complete -c fnclaude -s P -l from-pr        -d 'start from a PR (optional PR number or URL)'
complete -c fnclaude -s R -l remote-control -d 'enable remote control (optional name)'
complete -c fnclaude -s T -l tmux -a 'classic' -d 'set tmux mode (optional: classic)'

# -w / --worktree: complete existing worktree basenames from the current repo.
complete -c fnclaude -s w -l worktree -r -a '(__fnclaude_worktree_names)' -d 'use git worktree'

# ---------------------------------------------------------------------------
# Positional argument completion
# ---------------------------------------------------------------------------

# Helper: true when no positional has been typed yet (position 1).
function __fnclaude_no_positional
    set -l tokens (commandline -opc)
    set -l positionals 0
    set -l skip_next false
    for tok in $tokens[2..]
        if $skip_next
            set skip_next false
            continue
        end
        # Flags that consume the next token.
        if contains -- $tok --also -A --agent -G --permission-mode -M --allowedTools -W --tmux -T --from-pr -P --remote-control -R --worktree -w
            set skip_next true
            continue
        end
        if string match -q -- '-*' $tok
            continue
        end
        set positionals (math $positionals + 1)
    end
    test $positionals -eq 0
end

# Helper: true when exactly one positional has been typed and it was a model alias.
function __fnclaude_after_model
    set -l tokens (commandline -opc)
    set -l positionals 0
    set -l first_pos ''
    set -l skip_next false
    for tok in $tokens[2..]
        if $skip_next
            set skip_next false
            continue
        end
        if contains -- $tok --also -A --agent -G --permission-mode -M --allowedTools -W --tmux -T --from-pr -P --remote-control -R --worktree -w
            set skip_next true
            continue
        end
        if string match -q -- '-*' $tok
            continue
        end
        set positionals (math $positionals + 1)
        if test $positionals -eq 1
            set first_pos $tok
        end
    end
    test $positionals -eq 1 && contains -- $first_pos opus sonnet haiku
end

# Helper: true when two or more positionals have been typed (any further are dirs).
function __fnclaude_need_dir
    set -l tokens (commandline -opc)
    set -l positionals 0
    set -l skip_next false
    for tok in $tokens[2..]
        if $skip_next
            set skip_next false
            continue
        end
        if contains -- $tok --also -A --agent -G --permission-mode -M --allowedTools -W --tmux -T --from-pr -P --remote-control -R --worktree -w
            set skip_next true
            continue
        end
        if string match -q -- '-*' $tok
            continue
        end
        set positionals (math $positionals + 1)
    end
    test $positionals -ge 2
end

# Position 1: model alias.
complete -c fnclaude -n '__fnclaude_no_positional' -a 'opus'   -d 'use claude-opus model'
complete -c fnclaude -n '__fnclaude_no_positional' -a 'sonnet' -d 'use claude-sonnet model'
complete -c fnclaude -n '__fnclaude_no_positional' -a 'haiku'  -d 'use claude-haiku model'
# Position 1 can also be a directory.
complete -c fnclaude -n '__fnclaude_no_positional' -a '(__fish_complete_directories)' -d 'launch directory'

# Position 2: effort level (when pos1 was a model alias).
complete -c fnclaude -n '__fnclaude_after_model' -a 'low'    -d 'low effort'
complete -c fnclaude -n '__fnclaude_after_model' -a 'medium' -d 'medium effort'
complete -c fnclaude -n '__fnclaude_after_model' -a 'high'   -d 'high effort'
complete -c fnclaude -n '__fnclaude_after_model' -a 'xhigh'  -d 'extra-high effort'
complete -c fnclaude -n '__fnclaude_after_model' -a 'max'    -d 'maximum effort'
# Position 2 can also be a directory.
complete -c fnclaude -n '__fnclaude_after_model' -a '(__fish_complete_directories)' -d 'launch directory'

# Position 3+: directory only.
complete -c fnclaude -n '__fnclaude_need_dir' -a '(__fish_complete_directories)' -d 'extra directory'

# ---------------------------------------------------------------------------
# Register the same completions for the fnc alias.
# Fish doesn't support aliasing completion functions, so duplicate the key
# lines.  We re-use the same condition helpers since they're already defined.
# ---------------------------------------------------------------------------

complete -c fnc -f
complete -c fnc -n '__fish_use_subcommand' -l no-tmux       -d 'disable tmux integration'
complete -c fnc -n '__fish_use_subcommand' -l no-permissions -d 'disable auto dangerously-skip-permissions injection'
complete -c fnc -s B -l brief              -d 'brief output mode'
complete -c fnc -s C -l chrome             -d 'open in Chrome'
complete -c fnc -s D -l dangerously-skip-permissions -d 'skip all permission prompts'
complete -c fnc -s F -l fork-session       -d 'fork the current session'
complete -c fnc -s I -l ide                -d 'enable IDE integration'
complete -c fnc -s V -l verbose            -d 'verbose output'
complete -c fnc -s A -l also -r -a '(__fish_complete_directories)' -d 'add an extra directory'
complete -c fnc -s G -l agent -r           -d 'run a named agent'
complete -c fnc -s W -l allowedTools -r    -d 'restrict allowed tools'
complete -c fnc -s M -l permission-mode -r -a 'acceptEdits auto bypassPermissions default dontAsk plan' -d 'set permission mode'
complete -c fnc -s P -l from-pr            -d 'start from a PR (optional PR number or URL)'
complete -c fnc -s R -l remote-control     -d 'enable remote control (optional name)'
complete -c fnc -s T -l tmux -a 'classic'  -d 'set tmux mode (optional: classic)'
complete -c fnc -s w -l worktree -r -a '(__fnclaude_worktree_names)' -d 'use git worktree'
complete -c fnc -n '__fnclaude_no_positional' -a 'opus'   -d 'use claude-opus model'
complete -c fnc -n '__fnclaude_no_positional' -a 'sonnet' -d 'use claude-sonnet model'
complete -c fnc -n '__fnclaude_no_positional' -a 'haiku'  -d 'use claude-haiku model'
complete -c fnc -n '__fnclaude_no_positional' -a '(__fish_complete_directories)' -d 'launch directory'
complete -c fnc -n '__fnclaude_after_model' -a 'low medium high xhigh max' -d 'effort level'
complete -c fnc -n '__fnclaude_after_model' -a '(__fish_complete_directories)' -d 'launch directory'
complete -c fnc -n '__fnclaude_need_dir'    -a '(__fish_complete_directories)' -d 'extra directory'
