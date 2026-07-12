package atcoder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsAtcoderDirectory(t *testing.T) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { os.Chdir(originalDir) })

	tmp := t.TempDir()
	atcoderDir := filepath.Join(tmp, "atcoder")
	if err := os.Mkdir(atcoderDir, 0o755); err != nil {
		t.Fatalf("mkdir atcoder: %v", err)
	}
	otherDir := filepath.Join(tmp, "other")
	if err := os.Mkdir(otherDir, 0o755); err != nil {
		t.Fatalf("mkdir other: %v", err)
	}

	cases := []struct {
		name string
		dir  string
		want bool
	}{
		{"directory named atcoder", atcoderDir, true},
		{"directory not named atcoder", otherDir, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := os.Chdir(c.dir); err != nil {
				t.Fatalf("chdir %s: %v", c.dir, err)
			}
			if got := isAtcoderDirectory(); got != c.want {
				t.Errorf("isAtcoderDirectory() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestExtractTasks_BasicAbc375(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("testdata", "tasks_abc375.html"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	got := extractTasks(body, "abc375")
	if len(got) != 3 {
		t.Fatalf("len(extractTasks) = %d, want 3", len(got))
	}

	wantIDs := []string{"A", "B", "C"}
	wantTitleSubstrings := []string{"Frequency", "Traveling Takahashi Problem", "Spiral Rotation"}
	for i, p := range got {
		if p.ID != wantIDs[i] {
			t.Errorf("got[%d].ID = %q, want %q", i, p.ID, wantIDs[i])
		}
		if !strings.Contains(p.Title, wantTitleSubstrings[i]) {
			t.Errorf("got[%d].Title = %q, want substring %q", i, p.Title, wantTitleSubstrings[i])
		}
		if p.ContestID != "abc375" {
			t.Errorf("got[%d].ContestID = %q, want %q", i, p.ContestID, "abc375")
		}
		if !strings.HasPrefix(p.URL, atcoderHost+"/contests/abc375/") {
			t.Errorf("got[%d].URL = %q, want prefix %q", i, p.URL, atcoderHost+"/contests/abc375/")
		}
		if p.CurrentDate == "" {
			t.Errorf("got[%d].CurrentDate is empty", i)
		}
	}
}

func TestExtractTasks_EdgeCases(t *testing.T) {
	cases := []struct {
		name string
		body []byte
	}{
		{"empty body", []byte("")},
		{"no table", []byte(`<html><body><p>nothing here</p></body></html>`)},
		{"empty tbody", []byte(`<html><body><table><tbody></tbody></table></body></html>`)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := extractTasks(c.body, "abc"); len(got) != 0 {
				t.Errorf("extractTasks(%s) = %d problems, want 0", c.name, len(got))
			}
		})
	}
}

func TestExtractSamples_Basic(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("testdata", "problem_samples.html"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	samples, err := extractSamplesFromBody(body)
	if err != nil {
		t.Fatalf("extractSamplesFromBody: %v", err)
	}

	if len(samples) != 3 {
		t.Fatalf("len(samples) = %d, want 3", len(samples))
	}

	want := []struct {
		input  string
		output string
	}{
		{"3 5\n1 2 3\n<test>", "9\nhello world"},
		{"1 1\n100", "100"},
		{"2 4\n& &", "<result>"},
	}

	for i, s := range samples {
		if s.Input != want[i].input {
			t.Errorf("sample[%d].Input = %q, want %q", i, s.Input, want[i].input)
		}
		if s.Output != want[i].output {
			t.Errorf("sample[%d].Output = %q, want %q", i, s.Output, want[i].output)
		}
	}
}

func TestExtractSamples_NoSamples(t *testing.T) {
	body := []byte(`<html><body><h3>Constraints</h3><p>No samples here</p></body></html>`)
	samples, err := extractSamplesFromBody(body)
	if err != nil {
		t.Fatalf("extractSamplesFromBody: %v", err)
	}
	if len(samples) != 0 {
		t.Errorf("len(samples) = %d, want 0", len(samples))
	}
}

func TestWriteSourceFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "abc.rs")

	created, err := writeSourceFile([]byte("v1"), path, false)
	if err != nil {
		t.Fatalf("writeSourceFile create: %v", err)
	}
	if !created {
		t.Fatal("writeSourceFile create: want created=true")
	}

	skipped, err := writeSourceFile([]byte("v2"), path, false)
	if err != nil {
		t.Fatalf("writeSourceFile exists: %v", err)
	}
	if skipped {
		t.Fatal("writeSourceFile exists: want created=false")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(data) != "v1" {
		t.Errorf("content = %q, want %q", string(data), "v1")
	}

	overwritten, err := writeSourceFile([]byte("v2"), path, true)
	if err != nil {
		t.Fatalf("writeSourceFile force: %v", err)
	}
	if !overwritten {
		t.Fatal("writeSourceFile force: want created=true")
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file after force: %v", err)
	}
	if string(data) != "v2" {
		t.Errorf("content after force = %q, want %q", string(data), "v2")
	}
}

func TestRustContestHeader(t *testing.T) {
	got := rustContestHeader("abc466")
	wantSubstrings := []string{
		"// ABC466 - AtCoder Beginner Contest 466",
		"https://atcoder.jp/contests/abc466/tasks",
		"use proconio::input;",
	}
	for _, want := range wantSubstrings {
		if !strings.Contains(got, want) {
			t.Errorf("rustContestHeader() missing %q in:\n%s", want, got)
		}
	}
}

func TestEnsureModEntry(t *testing.T) {
	dir := t.TempDir()
	modPath := filepath.Join(dir, "mod.rs")
	if err := os.WriteFile(modPath, []byte("pub mod abc001;\n"), 0o644); err != nil {
		t.Fatalf("write mod.rs: %v", err)
	}
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { os.Chdir(originalDir) })
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	if err := ensureModEntry("abc002"); err != nil {
		t.Fatalf("ensureModEntry: %v", err)
	}
	data, err := os.ReadFile(modPath)
	if err != nil {
		t.Fatalf("read mod.rs: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "pub mod abc001;") {
		t.Errorf("mod.rs lost existing entry: %q", content)
	}
	if !strings.Contains(content, "pub mod abc002;") {
		t.Errorf("mod.rs missing new entry: %q", content)
	}

	if err := ensureModEntry("abc002"); err != nil {
		t.Fatalf("ensureModEntry duplicate: %v", err)
	}
	data, err = os.ReadFile(modPath)
	if err != nil {
		t.Fatalf("read mod.rs again: %v", err)
	}
	if strings.Count(string(data), "pub mod abc002;") != 1 {
		t.Errorf("mod.rs duplicated entry: %q", string(data))
	}
}
