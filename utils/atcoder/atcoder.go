package atcoder

import (
	"bytes"
	"errors"
	"fmt"
	"html"
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

	problemFileTempForGo = `package main

import "github.com/sa1la/goin"

// Solve{{.ID}} TODO {{.CurrentDate}} {{.ContestID}}.{{.ID}}
// {{.URL}}
func Solve{{.ID}}() {
	defer goin.Flush()
}
`

	problemFileTempForRust = `//TODO {{.CurrentDate}} {{.ContestID}}.{{.ID}} {{.Title}}
// {{.URL}}
#[allow(dead_code)]
pub fn solve_{{.ID}}() {
	input! {
		// from source
	}
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

type Sample struct {
	Input  string
	Output string
}

type langSpec struct {
	ext         string
	perProblem  bool
	formatter   string
	formatArgs  func(target string) []string
	transformID func(string) string
	header      string // written once at the top of single-file output
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
				header:      "use proconio::input;\n\n",
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

func CreateContestsTasks(contestID string, lang Lang, force bool) error {
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
	return createContestsProblems(problems, contestID, entry, force)
}

func fetchHTML(url, contestID string) ([]byte, error) {
	response, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, classifyHTTPError(response.StatusCode, contestID, url)
	}
	return io.ReadAll(io.LimitReader(response.Body, maxBodyBytes))
}

// classifyHTTPError maps AtCoder failure responses to actionable messages:
//   - 404 → contest not found / not started
//   - 429 → concurrent requests tripped the rate limiter
//   - 5xx → server-side issue
func classifyHTTPError(status int, contestID, url string) error {
	switch {
	case status == http.StatusNotFound:
		if contestID != "" {
			return fmt.Errorf("contest %s not found (not started yet?)", contestID)
		}
		return fmt.Errorf("page not found: %s", url)
	case status == http.StatusTooManyRequests:
		return fmt.Errorf("rate limited by AtCoder, try again: %s", url)
	case status >= 500:
		return fmt.Errorf("AtCoder server error (HTTP %d): %s", status, url)
	default:
		return fmt.Errorf("unexpected HTTP %d: %s", status, url)
	}
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

func ExtractSamples(problemURL, contestID string) ([]Sample, error) {
	body, err := fetchHTML(problemURL, contestID)
	if err != nil {
		return nil, fmt.Errorf("fetch problem page: %w", err)
	}
	return extractSamplesFromBody(body)
}

func extractSamplesFromBody(body []byte) ([]Sample, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parse problem page: %w", err)
	}

	var samples []Sample
	var currentInput string
	var currentOutput string

	doc.Find("h3").Each(func(i int, s *goquery.Selection) {
		text := strings.ToLower(s.Text())
		if strings.Contains(text, "sample input") {
			currentInput = ""
			s.NextFiltered("pre").Each(func(j int, pre *goquery.Selection) {
				currentInput = html.UnescapeString(pre.Text())
			})
		} else if strings.Contains(text, "sample output") {
			currentOutput = ""
			s.NextFiltered("pre").Each(func(j int, pre *goquery.Selection) {
				currentOutput = html.UnescapeString(pre.Text())
			})
			if currentOutput != "" {
				samples = append(samples, Sample{
					Input:  strings.TrimSpace(currentInput),
					Output: strings.TrimSpace(currentOutput),
				})
				currentInput = ""
				currentOutput = ""
			}
		}
	})

	return samples, nil
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

func writeSampleFile(data []byte, fileName string, force bool) (bool, error) {
	flags := os.O_WRONLY | os.O_CREATE
	if !force {
		flags |= os.O_EXCL
	}
	file, err := os.OpenFile(fileName, flags, 0o644)
	if err != nil {
		if !force && errors.Is(err, fs.ErrExist) {
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

func createContestsProblems(problems []Problem, contestID string, entry langEntry, force bool) error {
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
		if entry.spec.header != "" {
			content.WriteString(entry.spec.header)
		}
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

	if err := writeAllProblemSamples(problems, entry.spec.perProblem, force); err != nil {
		return err
	}

	if !created {
		return nil
	}
	return runFormatter(entry.spec.formatter, entry.spec.formatArgs(formatTarget))
}

func writeProblemSamples(prob Problem, perProblem bool, force bool) error {
	samples, err := ExtractSamples(prob.URL, prob.ContestID)
	if err != nil {
		return err
	}

	var testdataDir string
	if perProblem {
		testdataDir = filepath.Join(prob.ContestID, "testdata")
	} else {
		testdataDir = prob.ContestID
	}
	if err := os.MkdirAll(testdataDir, os.ModePerm); err != nil {
		return fmt.Errorf("mkdir %s: %w", testdataDir, err)
	}

	lowerID := strings.ToLower(prob.ID)
	for i, sample := range samples {
		n := i + 1
		inFile := filepath.Join(testdataDir, fmt.Sprintf("%s_%d.in", lowerID, n))
		outFile := filepath.Join(testdataDir, fmt.Sprintf("%s_%d.out", lowerID, n))

		if _, err := writeSampleFile([]byte(sample.Input), inFile, force); err != nil {
			return err
		}
		if _, err := writeSampleFile([]byte(sample.Output), outFile, force); err != nil {
			return err
		}
	}

	return nil
}

func writeAllProblemSamples(problems []Problem, perProblem bool, force bool) error {
	for _, prob := range problems {
		if err := writeProblemSamples(prob, perProblem, force); err != nil {
			return err
		}
	}
	return nil
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
	if filepath.Base(currentDir) == "atcoder" {
		return true
	}
	parent := filepath.Dir(currentDir)
	return filepath.Base(parent) == "atcoder"
}
