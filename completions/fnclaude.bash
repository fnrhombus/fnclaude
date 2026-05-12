# Bash completion for fnclaude (and its fnc alias).
#
# Install: source this file in ~/.bashrc, e.g.:
#   source /path/to/fnclaude.bash

_fnclaude_complete() {
    local cur prev words cword
    _init_completion || return

    # Model aliases (magic position 1).
    local -a model_aliases=(opus sonnet haiku)

    # Effort levels (magic position 2, only when position 1 was a model alias).
    local -a effort_levels=(low medium high xhigh max)

    # --permission-mode / -M enum values.
    local -a permission_modes=(acceptEdits auto bypassPermissions default dontAsk plan)

    # Flags that take a required argument (next word is the value).
    local -a flags_with_arg=(--also -A --agent -G --permission-mode -M --allowedTools -W)

    # Flags with optional arguments — greedy: complete the optional value only
    # when cur already follows the flag.
    local -a flags_with_opt=(--from-pr -P --remote-control -R --tmux -T --worktree -w)

    # Complete value for flags that take a required argument.
    case "$prev" in
        --also|-A)
            # Directory argument.
            _filedir -d
            return
            ;;
        --agent|-G|--allowedTools|-W)
            # Free-form string; no completion.
            return
            ;;
        --permission-mode|-M)
            COMPREPLY=( $(compgen -W "${permission_modes[*]}" -- "$cur") )
            return
            ;;
        --tmux|-T)
            COMPREPLY=( $(compgen -W "classic" -- "$cur") )
            return
            ;;
        # -w / --worktree is a claude passthrough flag.
        # TODO(worktree-completion): replace the empty return below with a call
        # to a helper that runs `git worktree list --porcelain` and extracts
        # worktree names, then feeds them to compgen.  Keep this case block as
        # the designated extension point.
        --worktree|-w)
            return
            ;;
        --from-pr|-P|--remote-control|-R)
            # Optional value; fall through to default completion below.
            ;;
    esac

    # If cur starts with '-', complete flags.
    if [[ "$cur" == -* ]]; then
        local -a all_flags=(
            --also --no-tmux --no-permissions
            --brief --chrome --dangerously-skip-permissions --fork-session
            --agent --ide --permission-mode --from-pr --remote-control
            --tmux --verbose --allowedTools --worktree
            -A -B -C -D -F -G -I -M -P -R -T -V -W -w
        )
        COMPREPLY=( $(compgen -W "${all_flags[*]}" -- "$cur") )
        return
    fi

    # Positional argument logic.
    # Count positionals seen so far (words that don't start with '-' and aren't
    # values consumed by a preceding flag).
    local positional_count=0
    local first_positional=""
    local i
    local skip_next=false

    for (( i=1; i < cword; i++ )); do
        local w="${words[i]}"

        if $skip_next; then
            skip_next=false
            continue
        fi

        # If this word is a flag that consumes the next token, skip the next.
        local f
        for f in "${flags_with_arg[@]}"; do
            if [[ "$w" == "$f" ]]; then
                skip_next=true
                break
            fi
        done
        $skip_next && continue

        # Skip flags and their inline =value forms.
        [[ "$w" == -* ]] && continue

        # It's a positional.
        (( positional_count++ ))
        if [[ $positional_count -eq 1 ]]; then
            first_positional="$w"
        fi
    done

    if [[ $positional_count -eq 0 ]]; then
        # Position 1: model alias or directory.
        COMPREPLY=( $(compgen -W "${model_aliases[*]}" -- "$cur") )
        _filedir -d
    elif [[ $positional_count -eq 1 ]]; then
        # Position 2: effort level (if pos1 was a model alias) or directory.
        local is_model=false
        local m
        for m in "${model_aliases[@]}"; do
            [[ "$first_positional" == "$m" ]] && is_model=true && break
        done
        if $is_model; then
            COMPREPLY+=( $(compgen -W "${effort_levels[*]}" -- "$cur") )
        fi
        _filedir -d
    else
        # Position 3+: directory only.
        _filedir -d
    fi
}

complete -F _fnclaude_complete fnclaude
complete -F _fnclaude_complete fnc
