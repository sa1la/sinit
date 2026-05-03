package runner

import (
	"os"
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
	src := `package abc375

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
		WorkDir:   contestDir,
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

	src := `package abc375

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
		WorkDir:   contestDir,
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

	src := `package abc375
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
		WorkDir:   contestDir,
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
	src := `package abc375
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
		WorkDir:   contestDir,
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
