package atcoder

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Lang string

const (
	LangGo   Lang = "go"
	LangRust Lang = "rust"

	atcoderHost  = "https://atcoder.jp"
	yesAnswer    = "y"
	httpTimeout  = 15 * time.Second
	maxBodyBytes = 4 << 20

	problemFileTempForGo = `package {{.ContestID}}

import "github.com/sa1la/goin"

//TODO {{.CurrentDate}} {{.ContestID}}.{{.ID}}
// {{.URL}}
func Solve{{.ID}}() {
	defer goin.Flush()

}`

	problemFileTempForRust = `//TODO {{.CurrentDate}} {{.ContestID}}.{{.ID}} {{.Title}}
// {{.URL}}
#[allow(dead_code)]
pub fn solve_{{.ID}}() {

}

`
)

type Problem struct {
	ContestID   string
	ID          string
	Title       string
	URL         string
	CurrentDate string
}

type langSpec struct {
	ext         string
	perProblem  bool
	formatter   string
	formatArgs  func(target string) []string
	transformID func(string) string
}

type langEntry struct {
	tmpl *template.Template
	spec langSpec
}

var (
	httpClient = &http.Client{Timeout: httpTimeout}
	registry   = map[Lang]langEntry{
		LangGo: {
			tmpl: template.Must(template.New("go").Parse(problemFileTempForGo)),
			spec: langSpec{
				ext:         "go",
				perProblem:  true,
				formatter:   "gofmt",
				formatArgs:  func(dir string) []string { return []string{"-w", dir} },
				transformID: func(s string) string { return s },
			},
		},
		LangRust: {
			tmpl: template.Must(template.New("rust").Parse(problemFileTempForRust)),
			spec: langSpec{
				ext:         "rs",
				perProblem:  false,
				formatter:   "rustfmt",
				formatArgs:  func(file string) []string { return []string{file} },
				transformID: strings.ToLower,
			},
		},
	}
)

func RenderProblem(w io.Writer, p Problem, lang Lang) error {
	entry, ok := registry[lang]
	if !ok {
		return fmt.Errorf("unsupported lang: %q", lang)
	}
	return entry.tmpl.Execute(w, p)
}

func CreateContestsTasks(contestID string, lang Lang) error {
	entry, ok := registry[lang]
	if !ok {
		return fmt.Errorf("unsupported lang: %q", lang)
	}
	url := fmt.Sprintf("%s/contests/%s/tasks?lang=en", atcoderHost, contestID)
	body, err := fetchHTML(url, contestID)
	if err != nil {
		return err
	}
	problems := extractTasks(body, contestID)
	return createContestsProblems(problems, contestID, entry)
}

func fetchHTML(url, contestID string) ([]byte, error) {
	response, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oops. %s not started yet", contestID)
	}
	return io.ReadAll(io.LimitReader(response.Body, maxBodyBytes))
}

func extractTasks(body []byte, contestID string) []Problem {
	problems := []Problem{}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return problems
	}
	currentDate := time.Now().Format("20060102")
	doc.Find("table tbody tr").Each(func(i int, s *goquery.Selection) {
		taskLink := s.Find("td a").Eq(0).AttrOr("href", "")
		taskID := s.Find("td a").Eq(0).Text()
		taskName := s.Find("td").Eq(1).Text()
		problems = append(problems, Problem{
			ID:          taskID,
			ContestID:   contestID,
			Title:       fmt.Sprintf("%s.%s", taskID, taskName),
			URL:         atcoderHost + taskLink,
			CurrentDate: currentDate,
		})
	})
	return problems
}

func createFile(data []byte, fileName string) (bool, error) {
	file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return false, nil
		}
		return false, fmt.Errorf("create %s: %w", fileName, err)
	}
	defer file.Close()
	if _, err := file.Write(data); err != nil {
		return false, fmt.Errorf("write %s: %w", fileName, err)
	}
	return true, nil
}

func createContestsProblems(problems []Problem, contestID string, entry langEntry) error {
	var formatTarget string
	created := false
	if entry.spec.perProblem {
		if err := os.MkdirAll(contestID, os.ModePerm); err != nil {
			return fmt.Errorf("mkdir %s: %w", contestID, err)
		}
		for _, prob := range problems {
			p := prob
			p.ID = entry.spec.transformID(prob.ID)
			fileName := filepath.Join(contestID, fmt.Sprintf("%s.%s", p.Title, entry.spec.ext))
			var data strings.Builder
			if err := entry.tmpl.Execute(&data, p); err != nil {
				return fmt.Errorf("render %s: %w", p.ID, err)
			}
			if c, err := createFile([]byte(data.String()), fileName); err != nil {
				return err
			} else if c {
				created = true
			}
		}
		formatTarget = contestID
	} else {
		fileName := fmt.Sprintf("%s.%s", contestID, entry.spec.ext)
		var content strings.Builder
		for _, prob := range problems {
			p := prob
			p.ID = entry.spec.transformID(prob.ID)
			if err := entry.tmpl.Execute(&content, p); err != nil {
				return fmt.Errorf("render %s: %w", p.ID, err)
			}
		}
		if c, err := createFile([]byte(content.String()), fileName); err != nil {
			return err
		} else if c {
			created = true
		}
		formatTarget = fileName
	}
	if !created {
		return nil
	}
	return runFormatter(entry.spec.formatter, entry.spec.formatArgs(formatTarget))
}

func runFormatter(name string, args []string) error {
	if name == "" {
		return nil
	}
	if _, err := exec.LookPath(name); err != nil {
		fmt.Fprintf(os.Stderr, "warning: %s not found in PATH, skipping format\n", name)
		return nil
	}
	if err := exec.Command(name, args...).Run(); err != nil {
		return fmt.Errorf("run %s: %w", name, err)
	}
	return nil
}

func CheckValidDir() bool {
	if isAtcoderDirectory() {
		return true
	}
	var userResponse string
	fmt.Print("not an Atcoder directory, continue? (y/n): ")
	fmt.Scanln(&userResponse)
	if userResponse != yesAnswer {
		fmt.Println("Exiting the program.")
		return false
	}
	return true
}

func isAtcoderDirectory() bool {
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current directory:", err)
		return false
	}
	return filepath.Base(currentDir) == "atcoder"
}
