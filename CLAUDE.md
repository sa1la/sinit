# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`sinit` is a Go CLI built with Cobra that automates two workflows:

1. **`sinit init`** — wipes a directory's `.git` history and creates a fresh repo with an initial commit.
2. **`sinit ac` / `sinit acr`** — scrapes AtCoder contest problems and scaffolds Go files per problem (`ac`) or a single Rust file (`acr`).
3. **`sinit run`** — compiles and runs solutions against stored sample inputs, comparing output.

## Development Commands

```bash
# Build the binary
go build -o sinit

# Run all tests
go test ./...

# Run a specific package's tests
go test ./utils/atcoder
go test ./utils/runner

# Run a single test
go test ./utils/atcoder -run TestExtractTasks_BasicAbc375

# Vet and format
go vet ./...
gofmt -w .
```

## Architecture

### Cobra Command Structure (`cmd/`)

Commands follow the standard Cobra pattern: each subcommand is a `*cobra.Command` defined in its own file, registered in `init()` via `rootCmd.AddCommand`. `main.go` delegates to `cmd.Execute()`.

- **`cmd/init.go`** — wraps `utils/project.InitProject`.
- **`cmd/atcoder.go`** — uses a `Lang`-typed factory (`newAtcoderCmd`) to produce the `ac` and `acr` subcommands. The only difference between them is the `atcoder.Lang` enum passed in.
- **`cmd/run.go`** — detects language by looking for `PROBLEM.*.go` (Go) or `CONTEST.rs` (Rust), then delegates to `runner.Run`. Contains the only CLI-facing language detection logic in the codebase.

### AtCoder Scaffolding (`utils/atcoder/`)

The core flow for `ac`/`acr`:

1. `CreateContestsTasks(contestID, lang, force)` fetches the contest tasks page (`/contests/{id}/tasks?lang=en`).
2. `extractTasks` parses the HTML with goquery, pulling problem IDs, titles, and URLs from the tasks table.
3. `createContestsProblems` renders templates per problem using the `registry` map keyed by `Lang`.
4. Concurrently, `writeAllProblemSamples` fetches each problem page and extracts Sample Input/Output blocks via `extractSamplesFromBody`.

**Lang Registry Pattern**

`utils/atcoder/atcoder.go` defines a `registry` map (`map[Lang]langEntry`) that unifies templates, file extensions, per-problem vs. single-file output, formatter commands, and ID transformations. Adding a new language means adding a new `Lang` constant and a new `langEntry` in the registry. Go emits one file per problem; Rust emits a single `CONTEST.rs` with all stubs.

**Sample Extraction**

Samples are discovered by scanning `<h3>` tags for "Sample Input" / "Sample Output" text and reading the next `<pre>` sibling. HTML entities **must** be decoded (`html.UnescapeString`) — un-decoded entities in extracted text was a prior bug (commit `22abb37`). Samples are written to `testdata/<problem>_<n>.in` and `.out`.

### Local Testing Runner (`utils/runner/`)

`runner.Run` discovers all `testdata/PROBLEM_*.in` files, compiles the solution, runs each input through the binary, and compares stdout against the corresponding `.out` file.

**Go Compilation Strategy**

Go solutions are compiled in a **temporary directory** with an ephemeral `go.mod`:

1. The source file is copied and its `package` declaration rewritten to `package main`.
2. A generated `main.go` wrapper calls `Solve{PROBLEM}()`.
3. `go mod init` + `go mod tidy` + `go build` runs in the temp directory.
4. The temp directory is cleaned up after execution.

This is necessary because the scaffolded files use a per-contest package name (e.g., `package abc375`) that isn't runnable as `main`.

**Rust Compilation Strategy**

Rust compiles the contest source file plus a generated `main.rs` that calls `solve_{problem}()` inside the contest module.

### Project Init (`utils/project/`)

Simple `git` command orchestration: `rm -rf .git`, then `git init`, `config`, `add .`, `commit -m "Initial commit"`.

## Testing

- `utils/atcoder/atcoder_test.go` — tests HTML parsing logic against committed fixture files in `utils/atcoder/testdata/`.
- `utils/runner/runner_test.go` — integration tests that create temp directories with source files and testdata, then invoke `Run`. Tests cover pass, fail, missing output, and timeout scenarios.
- **Go runtime is required** for runner tests (they invoke `go build`).

## Key Dependencies

- `github.com/spf13/cobra` — CLI framework.
- `github.com/PuerkitoBio/goquery` — HTML scraping for AtCoder pages.
- `github.com/sa1la/goin` — Go template uses this for I/O helpers; the ephemeral module will fetch it during compilation.

## File/Directory Conventions

- AtCoder commands expect to run inside a directory whose basename (or parent basename) is `atcoder`. `CheckValidDir` prompts for confirmation otherwise.
- Contest output goes into `./CONTESTID/` with `testdata/` inside it.
- Sample files: `testdata/{problem}_{n}.in` / `.out`. User-added custom cases can use any suffix matching `testdata/{problem}_*.in`.
