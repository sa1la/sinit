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
	var tmpDir string
	switch opts.Lang {
	case LangGo:
		binary, tmpDir, err = compileGo(opts)
	case LangRust:
		binary, tmpDir, err = compileRust(opts)
	default:
		err = fmt.Errorf("unsupported lang: %q", opts.Lang)
	}
	if err != nil {
		return nil, err
	}
	if tmpDir != "" {
		defer os.RemoveAll(tmpDir)
	}

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

func compileGo(opts Options) (string, string, error) {
	if _, err := exec.LookPath("go"); err != nil {
		return "", "", fmt.Errorf("go not found in PATH")
	}

	goPattern := filepath.Join(opts.WorkDir, opts.ContestID, strings.ToUpper(opts.ProblemID)+".*.go")
	matches, err := filepath.Glob(goPattern)
	if err != nil {
		return "", "", fmt.Errorf("glob source: %w", err)
	}
	if len(matches) == 0 {
		return "", "", fmt.Errorf("source file for problem %s not found in %s", opts.ProblemID, opts.WorkDir)
	}
	srcFile := matches[0]

	tmpDir, err := os.MkdirTemp("", "sinit-run-*")
	if err != nil {
		return "", "", fmt.Errorf("mkdirtemp: %w", err)
	}

	src, err := os.ReadFile(srcFile)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", "", fmt.Errorf("read source: %w", err)
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
		os.RemoveAll(tmpDir)
		return "", "", err
	}

	solveFunc := "Solve" + strings.ToUpper(opts.ProblemID)
	wrapper := fmt.Sprintf("package main\n\nfunc main() {\n\t%s()\n}\n", solveFunc)
	mainFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(mainFile, []byte(wrapper), 0o644); err != nil {
		os.RemoveAll(tmpDir)
		return "", "", err
	}

	modInit := exec.Command("go", "mod", "init", "sinit-run")
	modInit.Dir = tmpDir
	if out, err := modInit.CombinedOutput(); err != nil {
		os.RemoveAll(tmpDir)
		return "", "", fmt.Errorf("go mod init: %s", out)
	}

	modTidy := exec.Command("go", "mod", "tidy")
	modTidy.Dir = tmpDir
	if out, err := modTidy.CombinedOutput(); err != nil {
		os.RemoveAll(tmpDir)
		return "", "", fmt.Errorf("go mod tidy: %s", out)
	}

	compileCtx, compileCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer compileCancel()

	compileCmd := exec.CommandContext(compileCtx, "go", "build", "-o", "run", solveFile, mainFile)
	compileCmd.Dir = tmpDir

	if out, err := compileCmd.CombinedOutput(); err != nil {
		os.RemoveAll(tmpDir)
		if compileCtx.Err() == context.DeadlineExceeded {
			return "", "", fmt.Errorf("compilation timeout")
		}
		return "", "", fmt.Errorf("compile: %s", out)
	}

	return filepath.Join(tmpDir, "run"), tmpDir, nil
}

func compileRust(opts Options) (string, string, error) {
	if _, err := exec.LookPath("rustc"); err != nil {
		return "", "", fmt.Errorf("rustc not found in PATH")
	}

	srcFile := filepath.Join(opts.WorkDir, opts.ContestID+".rs")

	tmpDir, err := os.MkdirTemp("", "sinit-run-*")
	if err != nil {
		return "", "", fmt.Errorf("mkdirtemp: %w", err)
	}

	src, err := os.ReadFile(srcFile)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", "", fmt.Errorf("read source: %w", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, opts.ContestID+".rs"), src, 0o644); err != nil {
		os.RemoveAll(tmpDir)
		return "", "", err
	}

	solveFunc := "solve_" + strings.ToLower(opts.ProblemID)
	wrapper := fmt.Sprintf("mod %s;\n\nfn main() {\n\t%s::%s();\n}\n",
		opts.ContestID, opts.ContestID, solveFunc)
	if err := os.WriteFile(filepath.Join(tmpDir, "main.rs"), []byte(wrapper), 0o644); err != nil {
		os.RemoveAll(tmpDir)
		return "", "", err
	}

	compileCtx, compileCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer compileCancel()

	compileCmd := exec.CommandContext(compileCtx, "rustc", "main.rs", "-o", "run")
	compileCmd.Dir = tmpDir

	if out, err := compileCmd.CombinedOutput(); err != nil {
		os.RemoveAll(tmpDir)
		if compileCtx.Err() == context.DeadlineExceeded {
			return "", "", fmt.Errorf("compilation timeout")
		}
		return "", "", fmt.Errorf("compile: %s", out)
	}

	return filepath.Join(tmpDir, "run"), tmpDir, nil
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
