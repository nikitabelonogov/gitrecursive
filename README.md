# gitrecursive

Run a git command in every repository under the current directory — in parallel, with live per-repo progress.

```
⠹ backend        1.2s
✓ frontend       0.8s
⠧ services/infra 2.1s

── ✓ frontend ──────────────────────────
From github.com:me/frontend
   a1b2c3..d4e5f6  main -> origin/main
```

Each repo gets a spinner while it runs; its output is printed as a clean block when it finishes, followed by a summary table. In a non-interactive context (CI, pipes) the animation is dropped automatically and plain output is printed instead.

## Installation

```sh
go install github.com/nikitabelonogov/gitrecursive@latest
```

Or build from source:

```sh
git clone git@github.com:nikitabelonogov/gitrecursive.git
cd gitrecursive
go build -o gitrecursive .
```

## Usage

```sh
gitrecursive fetch --all
gitrecursive -j 4 pull --ff-only
gitrecursive --depth 1 status -sb
gitrecursive --timeout 30s --fail-fast fetch
gitrecursive --dry-run
```

Flags must come **before** the git command; everything after the first non-flag argument is passed to git verbatim (so `gitrecursive fetch --depth 2` sends `--depth 2` to git, while `gitrecursive --depth 2 fetch` limits repository discovery).

| Flag | Description |
| --- | --- |
| `-j, --jobs N` | Repositories processed in parallel (default: number of CPUs) |
| `-d, --depth N` | Limit discovery depth (`0` = current directory only, `-1` = unlimited) |
| `-t, --timeout D` | Per-repository timeout, e.g. `30s` (default: none) |
| `--fail-fast` | Cancel remaining repositories after the first failure |
| `--dry-run` | List discovered repositories without running anything |
| `--no-color` | Disable colored output (also respects `NO_COLOR`) |

Exits with code `0` when every repository succeeds and `1` when any fails or times out — handy in scripts and CI.

Because output is captured, interactive credential prompts are disabled (`GIT_TERMINAL_PROMPT=0`); repositories that would prompt fail instead of hanging.

## Shell completions

```sh
# zsh
gitrecursive completion zsh > "${fpath[1]}/_gitrecursive"

# bash
gitrecursive completion bash > /usr/local/etc/bash_completion.d/gitrecursive

# fish
gitrecursive completion fish > ~/.config/fish/completions/gitrecursive.fish
```

## Legacy bash version

The original sequential implementation lives in [`gitrecursive.sh`](gitrecursive.sh).
