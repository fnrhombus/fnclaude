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

# Token walker shared by all positional helpers. Sets three globals on
# return (caller-scope, via `set` without -l):
#   __fnclaude_magic_state         0=check model, 1=check effort, 2=magic done
#   __fnclaude_post_magic_count    count of post-magic, non-subcommand positionals
#   __fnclaude_first_pos           first post-magic positional (empty when none)
function __fnclaude_walk_tokens
    set -l tokens (commandline -opc)
    set -g __fnclaude_magic_state 0
    set -g __fnclaude_post_magic_count 0
    set -g __fnclaude_first_pos ''
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
        # Subcommand-style positionals: eat a slot without affecting magic or
        # the remaining-positional count.
        if contains -- $tok resume res continue con
            continue
        end
        # Magic at pos 1: model alias.
        if test $__fnclaude_magic_state -eq 0
            if contains -- $tok opus sonnet haiku
                set __fnclaude_magic_state 1
                continue
            end
            set __fnclaude_magic_state 2
        else if test $__fnclaude_magic_state -eq 1
            # Magic at pos 2: effort level (only after a model alias).
            if contains -- $tok low medium high xhigh max
                set __fnclaude_magic_state 2
                continue
            end
            set __fnclaude_magic_state 2
        end
        # Post-magic positional.
        set __fnclaude_post_magic_count (math $__fnclaude_post_magic_count + 1)
        if test $__fnclaude_post_magic_count -eq 1
            set __fnclaude_first_pos $tok
        end
    end
end

# Helper: at the position-1 slot (no post-magic positional typed AND magic not done).
function __fnclaude_pos_model
    __fnclaude_walk_tokens
    test $__fnclaude_magic_state -eq 0 -a $__fnclaude_post_magic_count -eq 0
end

# Helper: at the position-2 slot for effort (pos1 was a model alias).
function __fnclaude_pos_effort
    __fnclaude_walk_tokens
    test $__fnclaude_magic_state -eq 1 -a $__fnclaude_post_magic_count -eq 0
end

# Helper: at the cwd slot — no post-magic positional yet.
function __fnclaude_pos_cwd
    __fnclaude_walk_tokens
    test $__fnclaude_post_magic_count -eq 0
end

# Helper: at the worktree slot — exactly one post-magic positional.
function __fnclaude_pos_worktree
    __fnclaude_walk_tokens
    test $__fnclaude_post_magic_count -eq 1
end

# Helper: always (any positional slot may host a subcommand, max one).
function __fnclaude_any_positional_slot
    __fnclaude_walk_tokens
    test $__fnclaude_post_magic_count -le 1
end

# Position 1: model alias.
complete -c fnclaude -n '__fnclaude_pos_model' -a 'opus'   -d 'use claude-opus model'
complete -c fnclaude -n '__fnclaude_pos_model' -a 'sonnet' -d 'use claude-sonnet model'
complete -c fnclaude -n '__fnclaude_pos_model' -a 'haiku'  -d 'use claude-haiku model'

# Position 2: effort level (only when pos1 was a model alias).
complete -c fnclaude -n '__fnclaude_pos_effort' -a 'low'    -d 'low effort'
complete -c fnclaude -n '__fnclaude_pos_effort' -a 'medium' -d 'medium effort'
complete -c fnclaude -n '__fnclaude_pos_effort' -a 'high'   -d 'high effort'
complete -c fnclaude -n '__fnclaude_pos_effort' -a 'xhigh'  -d 'extra-high effort'
complete -c fnclaude -n '__fnclaude_pos_effort' -a 'max'    -d 'maximum effort'

# cwd slot (no post-magic positional typed yet): directory.
complete -c fnclaude -n '__fnclaude_pos_cwd' -a '(__fish_complete_directories)' -d 'launch directory'

# Worktree slot (one post-magic positional already typed): worktree basenames.
complete -c fnclaude -n '__fnclaude_pos_worktree' -a '(__fnclaude_worktree_names)' -d 'worktree name'

# Subcommands valid at any positional slot (max one per invocation; not
# enforced here — runtime catches the second).
complete -c fnclaude -n '__fnclaude_any_positional_slot' -a 'resume'   -d 'show session picker (--resume)'
complete -c fnclaude -n '__fnclaude_any_positional_slot' -a 'res'      -d 'show session picker (--resume)'
complete -c fnclaude -n '__fnclaude_any_positional_slot' -a 'continue' -d 'resume most recent session (--continue)'
complete -c fnclaude -n '__fnclaude_any_positional_slot' -a 'con'      -d 'resume most recent session (--continue)'

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
complete -c fnc -n '__fnclaude_pos_model' -a 'opus'   -d 'use claude-opus model'
complete -c fnc -n '__fnclaude_pos_model' -a 'sonnet' -d 'use claude-sonnet model'
complete -c fnc -n '__fnclaude_pos_model' -a 'haiku'  -d 'use claude-haiku model'
complete -c fnc -n '__fnclaude_pos_effort' -a 'low medium high xhigh max' -d 'effort level'
complete -c fnc -n '__fnclaude_pos_cwd' -a '(__fish_complete_directories)' -d 'launch directory'
complete -c fnc -n '__fnclaude_pos_worktree' -a '(__fnclaude_worktree_names)' -d 'worktree name'
complete -c fnc -n '__fnclaude_any_positional_slot' -a 'resume'   -d 'show session picker (--resume)'
complete -c fnc -n '__fnclaude_any_positional_slot' -a 'res'      -d 'show session picker (--resume)'
complete -c fnc -n '__fnclaude_any_positional_slot' -a 'continue' -d 'resume most recent session (--continue)'
complete -c fnc -n '__fnclaude_any_positional_slot' -a 'con'      -d 'resume most recent session (--continue)'
