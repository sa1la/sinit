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
