package runner

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRun_GoPass(t *testing.T) {
	tmp := t.TempDir()

	// Create contest directory structure
	contestDir := filepath.Join(tmp, "abc375")
	if err := os.MkdirAll(contestDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Create a simple Go source file
	src := `package main

import "fmt"

func SolveA() {
	var n int
	fmt.Scan(&n)
	fmt.Println(n * 2)
}
`
	srcFile := filepath.Join(contestDir, "A.Test.go")
	if err := os.WriteFile(srcFile, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	// Create testdata inside contest dir
	testdataDir := filepath.Join(contestDir, "testdata")
	if err := os.MkdirAll(testdataDir, 0o755); err != nil {
		t.Fatalf("mkdir testdata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testdataDir, "a_1.in"), []byte("5\n"), 0o644); err != nil {
		t.Fatalf("write in: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testdataDir, "a_1.out"), []byte("10\n"), 0o644); err != nil {
		t.Fatalf("write out: %v", err)
	}

	results, err := Run(Options{
		Lang:      LangGo,
		ContestID: "abc375",
		ProblemID: "a",
		WorkDir:   tmp,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}

	res := results[0]
	if !res.Passed {
		t.Errorf("Passed = false, want true; Actual=%q Expected=%q Error=%q",
			res.Actual, res.Expected, res.Error)
	}
	if res.SampleName != "a_1" {
		t.Errorf("SampleName = %q, want a_1", res.SampleName)
	}
}

func TestRun_GoFail(t *testing.T) {
	tmp := t.TempDir()

	contestDir := filepath.Join(tmp, "abc375")
	if err := os.MkdirAll(contestDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	src := `package main

import "fmt"

func SolveA() {
	var n int
	fmt.Scan(&n)
	fmt.Println(n * 3) // wrong answer
}
`
	if err := os.WriteFile(filepath.Join(contestDir, "A.Test.go"), []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	// Create testdata inside contest dir
	testdataDir := filepath.Join(contestDir, "testdata")
	if err := os.MkdirAll(testdataDir, 0o755); err != nil {
		t.Fatalf("mkdir testdata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testdataDir, "a_1.in"), []byte("5\n"), 0o644); err != nil {
		t.Fatalf("write in: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testdataDir, "a_1.out"), []byte("10\n"), 0o644); err != nil {
		t.Fatalf("write out: %v", err)
	}

	results, err := Run(Options{
		Lang:      LangGo,
		ContestID: "abc375",
		ProblemID: "a",
		WorkDir:   tmp,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}

	res := results[0]
	if res.Passed {
		t.Errorf("Passed = true, want false")
	}
	if res.Actual != "15" {
		t.Errorf("Actual = %q, want 15", res.Actual)
	}
	if res.Expected != "10" {
		t.Errorf("Expected = %q, want 10", res.Expected)
	}
}

func TestRun_GoMissingOut(t *testing.T) {
	tmp := t.TempDir()

	contestDir := filepath.Join(tmp, "abc375")
	if err := os.MkdirAll(contestDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	src := `package main
import "fmt"
func SolveA() { fmt.Println(42) }
`
	if err := os.WriteFile(filepath.Join(contestDir, "A.Test.go"), []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	// Create testdata inside contest dir
	testdataDir := filepath.Join(contestDir, "testdata")
	if err := os.MkdirAll(testdataDir, 0o755); err != nil {
		t.Fatalf("mkdir testdata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testdataDir, "a_1.in"), []byte("5\n"), 0o644); err != nil {
		t.Fatalf("write in: %v", err)
	}
	// Deliberately no .out file

	results, err := Run(Options{
		Lang:      LangGo,
		ContestID: "abc375",
		ProblemID: "a",
		WorkDir:   tmp,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}

	res := results[0]
	if res.Passed {
		t.Errorf("Passed = true, want false (missing output)")
	}
	if res.Error != "missing expected output file" {
		t.Errorf("Error = %q, want 'missing expected output file'", res.Error)
	}
}

func TestRun_GoTimeout(t *testing.T) {
	tmp := t.TempDir()

	contestDir := filepath.Join(tmp, "abc375")
	if err := os.MkdirAll(contestDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Infinite loop
	src := `package main
func SolveA() {
	for {}
}
`
	if err := os.WriteFile(filepath.Join(contestDir, "A.Test.go"), []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	// Create testdata inside contest dir
	testdataDir := filepath.Join(contestDir, "testdata")
	if err := os.MkdirAll(testdataDir, 0o755); err != nil {
		t.Fatalf("mkdir testdata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testdataDir, "a_1.in"), []byte("5\n"), 0o644); err != nil {
		t.Fatalf("write in: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testdataDir, "a_1.out"), []byte("10\n"), 0o644); err != nil {
		t.Fatalf("write out: %v", err)
	}

	results, err := Run(Options{
		Lang:      LangGo,
		ContestID: "abc375",
		ProblemID: "a",
		WorkDir:   tmp,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}

	res := results[0]
	if res.Passed {
		t.Errorf("Passed = true, want false")
	}
	if res.Error != "timeout" {
		t.Errorf("Error = %q, want timeout", res.Error)
	}
}

func TestRun_RustPass(t *testing.T) {
	if _, err := exec.LookPath("cargo"); err != nil {
		t.Skip("cargo not found in PATH")
	}

	tmp := t.TempDir()

	cargoToml := `[package]
name = "sinit-test"
version = "0.0.0"
edition = "2021"

[[bin]]
name = "sinit-test"
path = "src/main.rs"
`
	if err := os.WriteFile(filepath.Join(tmp, "Cargo.toml"), []byte(cargoToml), 0o644); err != nil {
		t.Fatalf("write Cargo.toml: %v", err)
	}

	srcDir := filepath.Join(tmp, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "main.rs"), []byte("fn main() {}\n"), 0o644); err != nil {
		t.Fatalf("write main.rs: %v", err)
	}

	src := `pub fn solve_a() {
	let mut input = String::new();
	std::io::stdin().read_line(&mut input).unwrap();
	let n: i32 = input.trim().parse().unwrap();
	println!("{}", n * 2);
}
`
	if err := os.WriteFile(filepath.Join(srcDir, "abc375.rs"), []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	testdataDir := filepath.Join(srcDir, "abc375")
	if err := os.MkdirAll(testdataDir, 0o755); err != nil {
		t.Fatalf("mkdir testdata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testdataDir, "a_1.in"), []byte("5\n"), 0o644); err != nil {
		t.Fatalf("write in: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testdataDir, "a_1.out"), []byte("10\n"), 0o644); err != nil {
		t.Fatalf("write out: %v", err)
	}

	results, err := Run(Options{
		Lang:      LangRust,
		ContestID: "abc375",
		ProblemID: "a",
		WorkDir:   srcDir,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}

	res := results[0]
	if !res.Passed {
		t.Errorf("Passed = false, want true; Actual=%q Expected=%q Error=%q",
			res.Actual, res.Expected, res.Error)
	}
	if res.SampleName != "a_1" {
		t.Errorf("SampleName = %q, want a_1", res.SampleName)
	}

	wrapperPath := filepath.Join(tmp, "examples", rustExampleName+".rs")
	if _, err := os.Stat(wrapperPath); !os.IsNotExist(err) {
		t.Errorf("wrapper %s not cleaned up after Run (err=%v)", wrapperPath, err)
	}
}

func TestBundleGo_StdlibOnly(t *testing.T) {
	tmp := t.TempDir()

	contestDir := filepath.Join(tmp, "abc375")
	if err := os.MkdirAll(contestDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	src := `package main

import "fmt"

func SolveA() {
	fmt.Println(42)
}
`
	if err := os.WriteFile(filepath.Join(contestDir, "A.Test.go"), []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	stageDir := t.TempDir()
	opts := Options{
		Lang:      LangGo,
		ContestID: "abc375",
		ProblemID: "a",
		WorkDir:   tmp,
	}

	bundlePath, err := BundleGo(opts, stageDir)
	if err != nil {
		t.Fatalf("BundleGo: %v", err)
	}

	if _, err := os.Stat(bundlePath); err != nil {
		t.Fatalf("bundle not found: %v", err)
	}

	// Verify the bundle compiles standalone.
	compileCmd := exec.Command("go", "build", bundlePath)
	if out, err := compileCmd.CombinedOutput(); err != nil {
		t.Fatalf("compile bundle: %v\n%s", err, out)
	}
}

func TestBundleGo_CopiesUserGoMod(t *testing.T) {
	tmp := t.TempDir()

	contestDir := filepath.Join(tmp, "abc375")
	if err := os.MkdirAll(contestDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	src := `package main

import "fmt"

func SolveA() {
	fmt.Println(42)
}
`
	if err := os.WriteFile(filepath.Join(contestDir, "A.Test.go"), []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	// Write a user go.mod in the work dir so BundleGo copies it.
	goMod := `module sinit-test

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	stageDir := t.TempDir()
	opts := Options{
		Lang:      LangGo,
		ContestID: "abc375",
		ProblemID: "a",
		WorkDir:   tmp,
	}

	bundlePath, err := BundleGo(opts, stageDir)
	if err != nil {
		t.Fatalf("BundleGo: %v", err)
	}

	if _, err := os.Stat(bundlePath); err != nil {
		t.Fatalf("bundle not found: %v", err)
	}

	// Verify the copied go.mod was staged.
	stagedGoMod := filepath.Join(stageDir, "go.mod")
	if _, err := os.Stat(stagedGoMod); err != nil {
		t.Fatalf("staged go.mod not found: %v", err)
	}
	data, err := os.ReadFile(stagedGoMod)
	if err != nil {
		t.Fatalf("read staged go.mod: %v", err)
	}
	if string(data) != goMod {
		t.Errorf("staged go.mod = %q, want %q", string(data), goMod)
	}
}

func TestBundleGo_MissingSource(t *testing.T) {
	tmp := t.TempDir()

	// No source file created — BundleGo should fail.
	stageDir := t.TempDir()
	opts := Options{
		Lang:      LangGo,
		ContestID: "abc375",
		ProblemID: "a",
		WorkDir:   tmp,
	}

	_, err := BundleGo(opts, stageDir)
	if err == nil {
		t.Fatal("BundleGo: expected error for missing source, got nil")
	}
}
