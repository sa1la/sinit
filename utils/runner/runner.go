package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/murosan/gollect"
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

	pattern := filepath.Join(testdataDir, strings.ToLower(opts.ProblemID)+"_*.in")
	inFiles, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob testdata: %w", err)
	}
	if len(inFiles) == 0 {
		return nil, fmt.Errorf("no testdata found; run `sinit ac` (Go) or `sinit acr` (Rust) to fetch samples")
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

	tmpDir, err := os.MkdirTemp("", "sinit-run-*")
	if err != nil {
		return "", nil, fmt.Errorf("mkdirtemp: %w", err)
	}
	cleanup := func() { os.RemoveAll(tmpDir) }

	bundlePath, err := BundleGo(opts, tmpDir)
	if err != nil {
		cleanup()
		return "", nil, err
	}

	buildDir := filepath.Join(tmpDir, "build")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("mkdir build: %w", err)
	}
	finalSrc := filepath.Join(buildDir, "main.go")
	if err := os.Rename(bundlePath, finalSrc); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("move bundle: %w", err)
	}

	compileCtx, compileCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer compileCancel()

	compileCmd := exec.CommandContext(compileCtx, "go", "build", "-o", "run", finalSrc)
	compileCmd.Dir = buildDir

	if out, err := compileCmd.CombinedOutput(); err != nil {
		cleanup()
		if compileCtx.Err() == context.DeadlineExceeded {
			return "", nil, fmt.Errorf("compilation timeout")
		}
		return "", nil, fmt.Errorf("compile: %s", out)
	}

	return filepath.Join(buildDir, "run"), cleanup, nil
}

// compileRust piggy-backs on the enclosing cargo project so external crates
// (proconio, etc.) resolve via the user's Cargo.toml instead of bare rustc,
// which has no access to dependencies.
func compileRust(opts Options) (string, func(), error) {
	if _, err := exec.LookPath("cargo"); err != nil {
		return "", nil, fmt.Errorf("cargo not found in PATH")
	}

	cargoRoot, err := findMarkerUpward(opts.WorkDir, "Cargo.toml")
	if err != nil {
		return "", nil, err
	}

	srcFile := filepath.Join(opts.WorkDir, opts.ContestID+".rs")

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

func findMarkerUpward(startDir, marker string) (string, error) {
	dir := startDir
	for {
		if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("%s not found above %s", marker, startDir)
		}
		dir = parent
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

var (
	cachedGoVersion string
	goVersionOnce   sync.Once
)

func goVersion() string {
	goVersionOnce.Do(func() {
		out, err := exec.Command("go", "env", "GOVERSION").Output()
		if err == nil {
			cachedGoVersion = strings.TrimPrefix(strings.TrimSpace(string(out)), "go")
		} else {
			cachedGoVersion = "1.21"
		}
	})
	return cachedGoVersion
}

func solveFuncName(opts Options) string {
	return "Solve" + strings.ToUpper(opts.ProblemID)
}

func writeMainGo(path, solveFunc string) error {
	wrapper := fmt.Sprintf("package main\n\nfunc main() {\n\t%s()\n}\n", solveFunc)
	return os.WriteFile(path, []byte(wrapper), 0o644)
}

func findGoSource(opts Options) (string, error) {
	goPattern := filepath.Join(opts.WorkDir, opts.ContestID, strings.ToUpper(opts.ProblemID)+".*.go")
	matches, err := filepath.Glob(goPattern)
	if err != nil {
		return "", fmt.Errorf("glob source: %w", err)
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("source file for problem %s not found in %s", opts.ProblemID, opts.WorkDir)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("ambiguous: multiple source files match for problem %s: %v", opts.ProblemID, matches)
	}
	return matches[0], nil
}

// stageGoSource locates the problem's Go source and copies it, along with a
// generated main.go wrapper, into dir.
func stageGoSource(opts Options, dir string) error {
	srcFile, err := findGoSource(opts)
	if err != nil {
		return err
	}
	src, err := os.ReadFile(srcFile)
	if err != nil {
		return fmt.Errorf("read source: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "solve.go"), src, 0o644); err != nil {
		return err
	}
	if err := writeMainGo(filepath.Join(dir, "main.go"), solveFuncName(opts)); err != nil {
		return err
	}
	return nil
}

// BundleGo generates a self-contained single-file Go bundle for the given
// problem using gollect. It stages solve.go, main.go and a go.mod inside
// stageDir, invokes gollect in-process, and returns the path to the produced
// bundle file.
//
// NOTE: BundleGo changes the process working directory and is NOT safe for
// concurrent use. Callers must ensure serialization.
func BundleGo(opts Options, stageDir string) (_ string, err error) {
	if err = stageGoSource(opts, stageDir); err != nil {
		return "", err
	}

	// Copy the user's go.mod/go.sum so gollect resolves the same module versions.
	// Note: replace directives in the user's go.mod may break gollect inlining.
	userGoMod, goModErr := findMarkerUpward(opts.WorkDir, "go.mod")
	if goModErr == nil {
		if err = copyFile(filepath.Join(userGoMod, "go.mod"), filepath.Join(stageDir, "go.mod")); err != nil {
			return "", fmt.Errorf("copy go.mod: %w", err)
		}
		if err = copyFile(filepath.Join(userGoMod, "go.sum"), filepath.Join(stageDir, "go.sum")); err != nil {
			if !os.IsNotExist(err) {
				return "", fmt.Errorf("copy go.sum: %w", err)
			}
		}
	} else {
		if err = os.WriteFile(filepath.Join(stageDir, "go.mod"), fmt.Appendf(nil, "module sinit-run\n\ngo %s\n", goVersion()), 0o644); err != nil {
			return "", err
		}
	}

	bundlePath := filepath.Join(stageDir, "bundle.go")

	// gollect.Main relies on packages.Load, which uses the process working
	// directory to locate go.mod. Switch to stageDir, restore on return.
	oldwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getwd: %w", err)
	}
	if err = os.Chdir(stageDir); err != nil {
		return "", fmt.Errorf("chdir stagedir: %w", err)
	}
	defer func() {
		if rerr := os.Chdir(oldwd); rerr != nil && err == nil {
			err = fmt.Errorf("restore working directory: %w", rerr)
		}
	}()

	cfg := gollect.DefaultConfig()
	cfg.InputFile = "*.go"
	cfg.OutputPaths = []string{bundlePath}

	var gollectErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				gollectErr = fmt.Errorf("gollect panicked: %v", r)
			}
		}()
		gollectErr = gollect.Main(cfg)
	}()
	if gollectErr != nil {
		return "", fmt.Errorf("gollect: %w", gollectErr)
	}

	return bundlePath, nil
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
