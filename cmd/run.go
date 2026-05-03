package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sa1la/sinit/utils/atcoder"
	"github.com/sa1la/sinit/utils/runner"
	"github.com/spf13/cobra"
)

const defaultContestPrefix = "abc"

// problemArgPattern matches combined -p forms:
//   "c"        -> letter only          (legacy)
//   "455c"     -> digits + letter      (apply default prefix)
//   "abc455c"  -> prefix + digits + letter
var problemArgPattern = regexp.MustCompile(`^([a-z]{3})?(\d+)?([a-z])$`)

// parseProblemArg splits a combined -p value into (contestID, problemID).
// Returns ("", letter) when the input is a bare letter so existing usage with
// an explicit -c still works. Bare-digits forms (e.g. "455c") assume
// defaultContestPrefix — wrong for arc/agc contests, so callers must let an
// explicit -c override.
//
// Examples:
//   "c"       -> ("",       "c")
//   "455c"    -> ("abc455", "c")
//   "abc455c" -> ("abc455", "c")
//   "arc183f" -> ("arc183", "f")
func parseProblemArg(raw string) (contestID, problemID string) {
	raw = strings.ToLower(raw)
	m := problemArgPattern.FindStringSubmatch(raw)
	if m == nil {
		return "", raw
	}
	prefix, digits, letter := m[1], m[2], m[3]
	if digits == "" {
		return "", letter
	}
	if prefix == "" {
		prefix = defaultContestPrefix
	}
	return prefix + digits, letter
}

func init() {
	var problem string
	var sample int
	var contestID string

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run solution against sample inputs",
		Long:  "Compile and run a solution against stored sample inputs, comparing output.",
		Run: func(cmd *cobra.Command, args []string) {
			if !atcoder.CheckValidDir() {
				return
			}

			wd, err := os.Getwd()
			if err != nil {
				fmt.Println("Error getting current directory:", err)
				return
			}

			if problem == "" {
				fmt.Println("Error: problem ID is required (use -p)")
				return
			}

			// Explicit -c always wins over the inferred value.
			inferred, prob := parseProblemArg(problem)
			problem = prob
			if contestID == "" {
				contestID = inferred
			}
			if contestID == "" {
				contestID = filepath.Base(wd)
			}

			// If already inside the contest directory, use wd; otherwise join it.
			var contestDir string
			if filepath.Base(wd) == contestID {
				contestDir = wd
			} else {
				contestDir = filepath.Join(wd, contestID)
			}

			lang := detectLang(wd, contestDir, contestID, problem)
			if lang == "" {
				fmt.Printf("Error: could not detect language for problem %s\n", problem)
				return
			}

			opts := runner.Options{
				Lang:      lang,
				ContestID: contestID,
				ProblemID: problem,
				WorkDir:   wd,
			}

			results, err := runner.Run(opts)
			if err != nil {
				fmt.Println("Error:", err)
				return
			}

			if sample > 0 {
				sampleName := fmt.Sprintf("%s_%d", strings.ToLower(problem), sample)
				var filtered []runner.Result
				for _, r := range results {
					if r.SampleName == sampleName {
						filtered = append(filtered, r)
						break
					}
				}
				results = filtered
				if len(results) == 0 {
					fmt.Printf("Error: sample %d not found for problem %s\n", sample, problem)
					return
				}
			}

			printResults(results)
		},
	}

	runCmd.Flags().StringVarP(&problem, "problem", "p", "", "problem ID (e.g., c, 455c, abc455c — bare digits assume abc prefix)")
	runCmd.Flags().IntVarP(&sample, "sample", "s", 0, "specific sample number (requires -p)")
	runCmd.Flags().StringVarP(&contestID, "contest", "c", "", "contest ID (defaults to current directory name)")

	rootCmd.AddCommand(runCmd)
}

func detectLang(wd, contestDir, contestID, problemID string) runner.Lang {
	goPattern := filepath.Join(contestDir, strings.ToUpper(problemID)+".*.go")
	matches, err := filepath.Glob(goPattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: glob pattern %q: %v\n", goPattern, err)
	}
	if len(matches) > 0 {
		return runner.LangGo
	}

	rsFile := filepath.Join(wd, contestID+".rs")
	if _, err := os.Stat(rsFile); err == nil {
		return runner.LangRust
	}

	return ""
}

func printResults(results []runner.Result) {
	fmt.Println()
	passed := 0
	for _, r := range results {
		if r.Error != "" {
			fmt.Printf("❌ %-12s FAILED (%s)\n", r.SampleName, r.Duration)
			fmt.Printf("   Error: %s\n", r.Error)
		} else if r.Passed {
			fmt.Printf("✅ %-12s passed (%s)\n", r.SampleName, r.Duration)
			passed++
		} else {
			fmt.Printf("❌ %-12s FAILED (%s)\n", r.SampleName, r.Duration)
			fmt.Printf("   Expected: %s\n", r.Expected)
			fmt.Printf("   Got:      %s\n", r.Actual)
		}
	}
	fmt.Println()
	fmt.Printf("Result: %d/%d passed\n", passed, len(results))
}
