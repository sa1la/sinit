package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Lang string

const (
	LangGo   Lang = "go"
	LangRust Lang = "rust"

	// Shared so wrapper filename, --example arg, and target binary name can't drift.
	rustExampleName = "sinit-run"
)

type Options struct {
	Lang      Lang
	ContestID string
	ProblemID string
	WorkDir   string
}

type Result struct {
	SampleName string
	Passed     bool
	Duration   time.Duration
	Actual     string
	Expected   string
	Error      string
}

// Run discovers all testdata samples for the problem and executes them.
func Run(opts Options) ([]Result, error) {
	var testdataDir string
	switch opts.Lang {
	case LangGo:
		testdataDir = filepath.Join(opts.WorkDir, opts.ContestID, "testdata")
	case LangRust:
		testdataDir = filepath.Join(opts.WorkDir, opts.ContestID)
	default:
		return nil, fmt.Errorf("unsupported lang: %q", opts.Lang)
	}
	if _, err := os.Stat(testdataDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("no testdata found; run `sinit ac` (Go) or `sinit acr` (Rust) to fetch samples")
	}

	pattern := filepath.Join(testdataDir, strings.ToLower(opts.ProblemID)+"_*.in")
	inFiles, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob testdata: %w", err)
	}
	if len(inFiles) == 0 {
		return nil, fmt.Errorf("no test inputs found for problem %s", opts.ProblemID)
	}

	var binary string
	var cleanup func()
	switch opts.Lang {
	case LangGo:
		binary, cleanup, err = compileGo(opts)
	case LangRust:
		binary, cleanup, err = compileRust(opts)
	default:
		err = fmt.Errorf("unsupported lang: %q", opts.Lang)
	}
	if err != nil {
		return nil, err
	}
	if cleanup != nil {
		defer cleanup()
	}

	// Prime the OS page cache so the first sample isn't penalized by cold-start.
	warmUp(binary, inFiles[0])

	var results []Result
	for _, inFile := range inFiles {
		sampleName := strings.TrimSuffix(filepath.Base(inFile), ".in")
		outFile := filepath.Join(testdataDir, sampleName+".out")

		res, err := runSample(binary, sampleName, inFile, outFile)
		if err != nil {
			return nil, err
		}
		results = append(results, res)
	}

	return results, nil
}

func runSample(binary, sampleName, inFile, outFile string) (Result, error) {
	expectedBytes, err := os.ReadFile(outFile)
	if err != nil {
		if os.IsNotExist(err) {
			return Result{
				SampleName: sampleName,
				Error:      "missing expected output file",
			}, nil
		}
		return Result{}, fmt.Errorf("read expected: %w", err)
	}

	actual, duration, runErr := execute(binary, inFile)

	res := Result{
		SampleName: sampleName,
		Duration:   duration,
		Expected:   strings.TrimSpace(string(expectedBytes)),
	}

	if runErr != nil {
		res.Error = runErr.Error()
		return res, nil
	}

	res.Actual = strings.TrimSpace(string(actual))
	res.Passed = res.Actual == res.Expected
	return res, nil
}

func compileGo(opts Options) (string, func(), error) {
	if _, err := exec.LookPath("go"); err != nil {
		return "", nil, fmt.Errorf("go not found in PATH")
	}

	goPattern := filepath.Join(opts.WorkDir, opts.ContestID, strings.ToUpper(opts.ProblemID)+".*.go")
	matches, err := filepath.Glob(goPattern)
	if err != nil {
		return "", nil, fmt.Errorf("glob source: %w", err)
	}
	if len(matches) == 0 {
		return "", nil, fmt.Errorf("source file for problem %s not found in %s", opts.ProblemID, opts.WorkDir)
	}
	srcFile := matches[0]

	tmpDir, err := os.MkdirTemp("", "sinit-run-*")
	if err != nil {
		return "", nil, fmt.Errorf("mkdirtemp: %w", err)
	}
	cleanup := func() { os.RemoveAll(tmpDir) }

	src, err := os.ReadFile(srcFile)
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("read source: %w", err)
	}
	srcStr := string(src)
	lines := strings.Split(srcStr, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "package ") {
			lines[i] = "package main"
			break
		}
	}
	rewritten := strings.Join(lines, "\n")
	solveFile := filepath.Join(tmpDir, "solve.go")
	if err := os.WriteFile(solveFile, []byte(rewritten), 0o644); err != nil {
		cleanup()
		return "", nil, err
	}

	solveFunc := "Solve" + strings.ToUpper(opts.ProblemID)
	wrapper := fmt.Sprintf("package main\n\nfunc main() {\n\t%s()\n}\n", solveFunc)
	mainFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(mainFile, []byte(wrapper), 0o644); err != nil {
		cleanup()
		return "", nil, err
	}

	modInit := exec.Command("go", "mod", "init", "sinit-run")
	modInit.Dir = tmpDir
	if out, err := modInit.CombinedOutput(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("go mod init: %s", out)
	}

	modTidy := exec.Command("go", "mod", "tidy")
	modTidy.Dir = tmpDir
	if out, err := modTidy.CombinedOutput(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("go mod tidy: %s", out)
	}

	compileCtx, compileCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer compileCancel()

	compileCmd := exec.CommandContext(compileCtx, "go", "build", "-o", "run", solveFile, mainFile)
	compileCmd.Dir = tmpDir

	if out, err := compileCmd.CombinedOutput(); err != nil {
		cleanup()
		if compileCtx.Err() == context.DeadlineExceeded {
			return "", nil, fmt.Errorf("compilation timeout")
		}
		return "", nil, fmt.Errorf("compile: %s", out)
	}

	return filepath.Join(tmpDir, "run"), cleanup, nil
}

// compileRust piggy-backs on the enclosing cargo project so external crates
// (proconio, etc.) resolve via the user's Cargo.toml instead of bare rustc,
// which has no access to dependencies.
func compileRust(opts Options) (string, func(), error) {
	if _, err := exec.LookPath("cargo"); err != nil {
		return "", nil, fmt.Errorf("cargo not found in PATH")
	}

	cargoRoot, err := findCargoRoot(opts.WorkDir)
	if err != nil {
		return "", nil, err
	}

	srcFile := filepath.Join(opts.WorkDir, opts.ContestID+".rs")
	if _, err := os.Stat(srcFile); err != nil {
		if os.IsNotExist(err) {
			return "", nil, fmt.Errorf("source file %s not found", srcFile)
		}
		return "", nil, fmt.Errorf("stat source: %w", err)
	}

	examplesDir := filepath.Join(cargoRoot, "examples")
	if err := os.MkdirAll(examplesDir, 0o755); err != nil {
		return "", nil, fmt.Errorf("mkdir examples: %w", err)
	}
	wrapperPath := filepath.Join(examplesDir, rustExampleName+".rs")

	solveFunc := "solve_" + strings.ToLower(opts.ProblemID)
	wrapper := fmt.Sprintf(`#[path = %q]
mod %s;

fn main() {
    %s::%s();
}
`, srcFile, opts.ContestID, opts.ContestID, solveFunc)
	if err := os.WriteFile(wrapperPath, []byte(wrapper), 0o644); err != nil {
		return "", nil, fmt.Errorf("write wrapper: %w", err)
	}
	cleanup := func() { os.Remove(wrapperPath) }

	compileCtx, compileCancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer compileCancel()

	compileCmd := exec.CommandContext(compileCtx, "cargo", "build", "--example", rustExampleName)
	compileCmd.Dir = cargoRoot

	if out, err := compileCmd.CombinedOutput(); err != nil {
		cleanup()
		if compileCtx.Err() == context.DeadlineExceeded {
			return "", nil, fmt.Errorf("compilation timeout")
		}
		return "", nil, fmt.Errorf("compile: %s", out)
	}

	binary := filepath.Join(cargoRoot, "target", "debug", "examples", rustExampleName)
	return binary, cleanup, nil
}

func findCargoRoot(startDir string) (string, error) {
	dir := startDir
	for {
		if _, err := os.Stat(filepath.Join(dir, "Cargo.toml")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("Cargo.toml not found above %s", startDir)
		}
		dir = parent
	}
}

// warmUp runs the binary once and discards the result, priming the OS page
// cache and dyld so the first timed sample isn't 100x slower than the rest
// due to cold-start overhead. Errors are swallowed — if the binary is broken,
// the timed runs that follow will surface the error.
func warmUp(binary, inFile string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary)
	cmd.Dir = filepath.Dir(binary)

	in, err := os.Open(inFile)
	if err != nil {
		return
	}
	defer in.Close()
	cmd.Stdin = in

	_ = cmd.Run()
}

func execute(binary, inFile string) ([]byte, time.Duration, error) {
	runCtx, runCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer runCancel()

	runCmd := exec.CommandContext(runCtx, binary)
	runCmd.Dir = filepath.Dir(binary)

	in, err := os.Open(inFile)
	if err != nil {
		return nil, 0, fmt.Errorf("open input: %w", err)
	}
	defer in.Close()
	runCmd.Stdin = in

	start := time.Now()
	out, err := runCmd.Output()
	duration := time.Since(start)

	if err != nil {
		if runCtx.Err() == context.DeadlineExceeded {
			return nil, duration, fmt.Errorf("timeout")
		}
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
			return nil, duration, fmt.Errorf("%s", exitErr.Stderr)
		}
		return nil, duration, err
	}

	return out, duration, nil
}
