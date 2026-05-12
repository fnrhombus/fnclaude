# fnclaude shell completions

Completion scripts for `fnclaude` and its `fnc` alias. Pick the file for your shell.

## zsh — `_fnclaude`

`_fnclaude` is a `compdef`-style completion function that covers both `fnclaude` and `fnc`.

**Option A — drop into `$fpath`:**

```sh
cp _fnclaude /usr/local/share/zsh/site-functions/   # or any dir in $fpath
# restart shell or run: autoload -U compinit && compinit
```

**Option B — source directly:**

```sh
# in ~/.zshrc
source /path/to/completions/_fnclaude
```

## bash — `fnclaude.bash`

Registers `_fnclaude_complete` for both `fnclaude` and `fnc` via `complete -F`.

```sh
# in ~/.bashrc
source /path/to/completions/fnclaude.bash
```

Requires bash-completion (`_init_completion`) to be loaded first. Most distros do this automatically; if not, add `source /usr/share/bash-completion/bash_completion` before the source line above.

## fish — `fnclaude.fish`

```sh
cp fnclaude.fish ~/.config/fish/completions/
```

Fish auto-loads files from that directory. The file registers completions for both `fnclaude` and `fnc`.
