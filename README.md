# sinit

A small CLI that automates two recurring chores when starting projects:

1. **`sinit init`** — wipes the target directory's `.git` history and creates a fresh repo with an initial commit.
2. **`sinit ac` / `sinit acr`** — pulls problems for an AtCoder contest and scaffolds a Go file per problem (`ac`) or a single Rust file with all stubs (`acr`).

## Install

```bash
go install github.com/sa1la/sinit@latest
```

Or build from source:

```bash
git clone https://github.com/sa1la/sinit.git
cd sinit
go build -o sinit
```

## Usage

### Reset / re-initialize a git repository

```bash
sinit init -p ./my-project -u alice -e alice@example.com
```

| flag | default | meaning |
|---|---|---|
| `-p`, `--project` | `.` | path to the project directory |
| `-u`, `--user`    | `admin` | local `user.name` |
| `-e`, `--email`   | `default@email.com` | local `user.email` |

This deletes `<project>/.git`, runs `git init`, sets a local user, stages everything, and creates an `Initial commit`.

### Scaffold an AtCoder contest

```bash
cd /path/to/atcoder

# Go: one .go file per problem inside ./<contestID>/
sinit ac -c abc375

# Rust: a single ./<contestID>.rs with one stub fn per problem
sinit acr -c abc375
```

Both subcommands look at the current directory's name; if it does not end in `atcoder`, they prompt before continuing.

The Go template depends on [`github.com/sa1la/goin`](https://github.com/sa1la/goin) and is formatted with `gofmt` (skipped if not on PATH). The Rust output is formatted with `rustfmt` (also optional). Both templates live in `utils/atcoder/atcoder.go`.

## Requirements

- Go 1.23+
- `git` on PATH (for `init`)
- `gofmt` on PATH (optional; `ac` will skip formatting if missing)
- `rustfmt` on PATH (optional; `acr` will skip formatting if missing)
