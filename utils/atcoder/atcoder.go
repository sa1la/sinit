package atcoder

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"

	"net/http"

	"github.com/PuerkitoBio/goquery"
)

const (
	_Atcoder_host = "https://atcoder.jp"

	_Yes = "y"

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
const GOLANG = "go"

const RUST = "rust"

// Problem represents a coding problem in an AtCoder contest.
type Problem struct {
	ContestID   string // ContestID identifies the specific contest this problem belongs to.
	ID          string // ID is the unique identifier for the problem within the contest.
	Title       string // Title is the name of the problem.
	URL         string // URL is the link to the problem's page on the AtCoder website.
	CurrentDate string
}

var contest string

func CreateContestsTasks(contestID string, lang string) error {

	contest = contestID
	url := fmt.Sprintf("https://atcoder.jp/contests/%s/tasks?lang=en", contestID)
	body, err := fetchHTML(url)
	if err != nil {
		return err
	}
	problems := extractTasks(body)
	if lang == GOLANG {
		createContestsProblemsForGo(problems, contestID)
	} else {
		createContestsProblemsForRust(problems, contestID)
	}
	return nil
}

func fetchHTML(url string) ([]byte, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oops. %s not started yet", contest)
	}
	html, err := io.ReadAll(response.Body)
	return html, err
}

// 获取题目信息
func extractTasks(body []byte) []Problem {
	problems := []Problem{}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return problems
	}
	// 查找任务表格并提取信息
	doc.Find("table tbody tr").Each(func(i int, s *goquery.Selection) {
		taskLink := s.Find("td a").Eq(0).AttrOr("href", "")
		taskID := s.Find("td a").Eq(0).Text()
		taskName := s.Find("td").Eq(1).Text()
		currentDate := time.Now().Format("20060102")
		problems = append(problems, Problem{ID: taskID, ContestID: contest, Title: fmt.Sprintf("%s.%s", taskID, taskName), URL: fmt.Sprintf("%s%s", _Atcoder_host, taskLink), CurrentDate: currentDate})
	})
	return problems
}

func createFile(data []byte, fileName string) error {
	_, err := os.Stat(fileName)
	if !os.IsNotExist(err) {
		return nil
	}
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("error creating file %s: %v", fileName, err)
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("error writing to file %s: %v", fileName, err)
	}

	return nil
}

func createContestsProblemsForGo(problems []Problem, contestsID string) error {
	if err := os.MkdirAll(contestsID, os.ModePerm); err != nil {
		return fmt.Errorf("error creating directory %s: %v", contestsID, err)
	}

	// Change directory and defer to recover it later
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error getting current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(contestsID); err != nil {
		return fmt.Errorf("error changing directory to %s: %v", contestsID, err)
	}

	for _, prob := range problems {
		fileName := fmt.Sprintf("%s.go", prob.Title)

		var data strings.Builder
		tmpl, err := template.New("problem").Parse(_ProblemFileTempForGo)
		if err != nil {
			return fmt.Errorf("error parsing template: %v", err)
		}
		if err := tmpl.Execute(&data, prob); err != nil {
			return fmt.Errorf("error executing template: %v", err)
		}

		if err := createFile([]byte(data.String()), fileName); err != nil {
			return fmt.Errorf("error creating problem file %s: %v", fileName, err)
		}
	}

	cmd := exec.Command("gofmt", "-w", ".")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running gofmt: %v", err)
	}

	return nil
}

func createContestsProblemsForRust(problems []Problem, contestsID string) error {
	fileName := fmt.Sprintf("%s.rs", contestsID)
	fileContent := []byte{}
	for _, prob := range problems {
		var data strings.Builder
		tmpl, err := template.New("problem").Parse(_ProblemFileTempForRust)
		if err != nil {
			return fmt.Errorf("error parsing template: %v", err)
		}
		prob.ID = strings.ToLower(prob.ID)
		if err := tmpl.Execute(&data, prob); err != nil {
			return fmt.Errorf("error executing template: %v", err)
		}
		fileContent = append(fileContent, []byte(data.String())...)
	}
	if err := createFile(fileContent, fileName); err != nil {
		return fmt.Errorf("error creating problem file %s: %v", fileName, err)
	}

	cmd := exec.Command("rustfmt", fileName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running gofmt: %v", err)
	}

	return nil
}

func CheckValidDir() {
	if !isAtcoderDirectory() {
		var userResponse string
		fmt.Print("not an Atcoder directory, continue? (y/n): ")
		fmt.Scanln(&userResponse)

		if userResponse != _Yes {
			fmt.Println("Exiting the program.")
			os.Exit(0)
		}
	}

}

func isAtcoderDirectory() bool {
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current directory:", err)
		return false
	}

	return strings.HasSuffix(currentDir, "/atcoder")
}
