package atcoder

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	_AtcoderHost = "https://atcoder.jp"
	_Yes         = "y"
	_HTTPTimeout = 15 * time.Second

	GOLANG = "go"
	RUST   = "rust"

	_ProblemFileTempForGo = `package {{.ContestID}}

import "github.com/sa1la/goin"

//TODO {{.CurrentDate}} {{.ContestID}}.{{.ID}}
// {{.URL}}
func Solve{{.ID}}() {
	defer goin.Flush()

}`

	_ProblemFileTempForRust = `//TODO {{.CurrentDate}} {{.ContestID}}.{{.ID}} {{.Title}}
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

var (
	httpClient = &http.Client{Timeout: _HTTPTimeout}
	goTmpl     = template.Must(template.New("go").Parse(_ProblemFileTempForGo))
	rustTmpl   = template.Must(template.New("rust").Parse(_ProblemFileTempForRust))
)

func RenderProblem(w io.Writer, p Problem, lang string) error {
	switch lang {
	case GOLANG:
		return goTmpl.Execute(w, p)
	case RUST:
		return rustTmpl.Execute(w, p)
	default:
		return fmt.Errorf("unsupported lang: %q", lang)
	}
}

func CreateContestsTasks(contestID, lang string) error {
	url := fmt.Sprintf("%s/contests/%s/tasks?lang=en", _AtcoderHost, contestID)
	body, err := fetchHTML(url, contestID)
	if err != nil {
		return err
	}
	problems := extractTasks(body, contestID)
	switch lang {
	case GOLANG:
		return createContestsProblemsForGo(problems, contestID)
	case RUST:
		return createContestsProblemsForRust(problems, contestID)
	default:
		return fmt.Errorf("unsupported lang: %q", lang)
	}
}

func fetchHTML(url, contestID string) ([]byte, error) {
	response, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oops. %s not started yet", contestID)
	}
	return io.ReadAll(response.Body)
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
			URL:         _AtcoderHost + taskLink,
			CurrentDate: currentDate,
		})
	})
	return problems
}

func createFile(data []byte, fileName string) error {
	if _, err := os.Stat(fileName); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("error stating file %s: %v", fileName, err)
	}

	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("error creating file %s: %v", fileName, err)
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("error writing to file %s: %v", fileName, err)
	}
	return nil
}

func createContestsProblemsForGo(problems []Problem, contestID string) error {
	if err := os.MkdirAll(contestID, os.ModePerm); err != nil {
		return fmt.Errorf("error creating directory %s: %v", contestID, err)
	}

	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error getting current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(contestID); err != nil {
		return fmt.Errorf("error changing directory to %s: %v", contestID, err)
	}

	for _, prob := range problems {
		fileName := fmt.Sprintf("%s.go", prob.Title)

		var data strings.Builder
		if err := RenderProblem(&data, prob, GOLANG); err != nil {
			return fmt.Errorf("error rendering problem %s: %v", prob.ID, err)
		}

		if err := createFile([]byte(data.String()), fileName); err != nil {
			return fmt.Errorf("error creating problem file %s: %v", fileName, err)
		}
	}

	if _, err := exec.LookPath("gofmt"); err == nil {
		if err := exec.Command("gofmt", "-w", ".").Run(); err != nil {
			return fmt.Errorf("error running gofmt: %v", err)
		}
	}
	return nil
}

func createContestsProblemsForRust(problems []Problem, contestID string) error {
	fileName := fmt.Sprintf("%s.rs", contestID)
	var content strings.Builder
	for _, prob := range problems {
		prob.ID = strings.ToLower(prob.ID)
		if err := RenderProblem(&content, prob, RUST); err != nil {
			return fmt.Errorf("error rendering problem %s: %v", prob.ID, err)
		}
	}
	if err := createFile([]byte(content.String()), fileName); err != nil {
		return fmt.Errorf("error creating problem file %s: %v", fileName, err)
	}

	if _, err := exec.LookPath("rustfmt"); err == nil {
		if err := exec.Command("rustfmt", fileName).Run(); err != nil {
			return fmt.Errorf("error running rustfmt: %v", err)
		}
	}
	return nil
}

func CheckValidDir() {
	if isAtcoderDirectory() {
		return
	}
	var userResponse string
	fmt.Print("not an Atcoder directory, continue? (y/n): ")
	fmt.Scanln(&userResponse)
	if userResponse != _Yes {
		fmt.Println("Exiting the program.")
		os.Exit(0)
	}
}

func isAtcoderDirectory() bool {
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current directory:", err)
		return false
	}
	return filepath.Base(currentDir) == "atcoder"
}
