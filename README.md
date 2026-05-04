# sinit

A small CLI for the chores around competitive programming (mostly AtCoder) and fresh project repos:

1. **`sinit init`** — wipe a directory's `.git` history and create a fresh repo with an initial commit.
2. **`sinit ac` / `sinit acr`** — pull problems for an AtCoder contest and scaffold solution files (Go: one file per problem; Rust: a single source file with all stubs).
3. **`sinit run`** — compile and run a solution against the stored sample inputs.
4. **`sinit bundle`** — produce a self-contained, single-file Go source ready to paste into AtCoder's submission box.

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

### `sinit init` — reset a project's git history

```bash
sinit init -p ./my-project -u alice -e alice@example.com
```

| flag | default | meaning |
|---|---|---|
| `-p`, `--project` | `.` | path to the project directory |
| `-u`, `--user`    | `admin` | local `user.name` |
| `-e`, `--email`   | `default@email.com` | local `user.email` |

Deletes `<project>/.git`, runs `git init`, sets the local user, stages everything, and creates `Initial commit`.

### `sinit ac` / `sinit acr` — scaffold an AtCoder contest

```bash
cd /path/to/atcoder

# Go: one .go file per problem, inside ./<contestID>/
sinit ac -c abc375

# Rust: a single ./<contestID>.rs with one stub fn per problem
sinit acr -c abc375
```

| flag | default | meaning |
|---|---|---|
| `-c`, `--contest` | *(prompted)* | contest ID, e.g. `abc375`, `arc183` |
| `-f`, `--force`   | `false` | overwrite existing sample files |

Both subcommands check that the current directory — or its immediate parent — is named `atcoder`; otherwise they prompt before continuing. Sample inputs/outputs are fetched concurrently and written to:

- Go:   `./<contestID>/testdata/<problem>_<n>.{in,out}`
- Rust: `./<contestID>/<problem>_<n>.{in,out}`

The Go template depends on [`github.com/sa1la/goin`](https://github.com/sa1la/goin) (auto-fetched on first compile). Output is run through `gofmt` / `rustfmt` if available — skipped silently if not.

### `sinit run` — run a solution against the samples

```bash
# All three forms select problem C of abc455:
sinit run -p abc455c
sinit run -p 455c            # bare digits assume the abc prefix
sinit run -c abc455 -p c     # explicit contest, legacy form

# Run only sample 2:
sinit run -p abc455c -s 2
```

| flag | default | meaning |
|---|---|---|
| `-p`, `--problem` | *(required)* | problem ID — accepts `c`, `455c`, or `abc455c` |
| `-c`, `--contest` | *(inferred)* | overrides the contest inferred from `-p` or the cwd |
| `-s`, `--sample`  | `0` | run a single sample by number instead of all |

Language is auto-detected: if `<contest>/<PROBLEM>.*.go` exists it's Go, otherwise it falls back to `<contest>.rs`. Each sample is timed and compared line-by-line; results are printed with ✅ / ❌. Running a Rust solution additionally requires a `Cargo.toml` somewhere up the directory tree — `sinit` builds the wrapper as a `cargo --example` inside that workspace.

### `sinit bundle` — build a single-file Go submission

```bash
# Print to stdout
sinit bundle -p abc455c

# Write to ./submit.go (default file when -o has no value)
sinit bundle -p abc455c -o

# Write to a custom path (must use "=" — space-separated form is not accepted)
sinit bundle -p abc455c -o=submission.go

# Copy straight to the clipboard
sinit bundle -p abc455c --copy
```

| flag | default | meaning |
|---|---|---|
| `-p`, `--problem` | *(required)* | problem ID, same syntax as `run` |
| `-c`, `--contest` | *(inferred)* | overrides the contest inferred from `-p` or the cwd |
| `-o`, `--output`  | *(stdout)* | write the bundle to a file; `-o` alone uses `submit.go`, `-o=NAME` uses `NAME` |
| `--copy`          | `false` | copy the bundle to the system clipboard |

Inlines the `goin` helper library (and any other internal imports) into a single source file using [`gollect`](https://github.com/murosan/gollect), which is vendored as a library — no separate binary to install.

Clipboard backend: `pbcopy` (macOS), `clip` (Windows), or `xclip` / `xsel` (Linux — install one of them).

## Requirements

- Go 1.25+
- `git` on PATH (for `init`)
- A Rust toolchain with `cargo` on PATH, plus an enclosing `Cargo.toml` (only for `run` on Rust solutions; `acr` itself only needs `rustfmt`)
- `gofmt` / `rustfmt` on PATH (optional; formatting is skipped if missing)
- `pbcopy` / `clip` / `xclip` / `xsel` (only when using `bundle --copy`)
