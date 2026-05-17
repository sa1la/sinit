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
//
//	"c"        -> letter only          (legacy)
//	"455c"     -> digits + letter      (apply default prefix)
//	"abc455c"  -> prefix + digits + letter
var problemArgPattern = regexp.MustCompile(`^([a-z]{3})?(\d+)?([a-z])$`)

// parseProblemArg splits a combined -p value into (contestID, problemID).
// Returns ("", letter) when the input is a bare letter so existing usage with
// an explicit -c still works. Bare-digits forms (e.g. "455c") assume
// defaultContestPrefix — wrong for arc/agc contests, so callers must let an
// explicit -c override.
//
// Examples:
//
//	"c"       -> ("",       "c")
//	"455c"    -> ("abc455", "c")
//	"abc455c" -> ("abc455", "c")
//	"arc183f" -> ("arc183", "f")
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

			contestRoot, cid, problemID := resolveContest(wd, problem, contestID)

			lang, _ := detectLang(contestRoot, cid, problemID)
			if lang == "" {
				fmt.Printf("Error: could not detect language for problem %s\n", problemID)
				return
			}

			opts := runner.Options{
				Lang:      lang,
				ContestID: cid,
				ProblemID: problemID,
				WorkDir:   contestRoot,
			}

			results, err := runner.Run(opts)
			if err != nil {
				fmt.Println("Error:", err)
				return
			}

			if sample > 0 {
				sampleName := fmt.Sprintf("%s_%d", strings.ToLower(problemID), sample)
				var filtered []runner.Result
				for _, r := range results {
					if r.SampleName == sampleName {
						filtered = append(filtered, r)
						break
					}
				}
				results = filtered
				if len(results) == 0 {
					fmt.Printf("Error: sample %d not found for problem %s\n", sample, problemID)
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

// resolveContest reconciles the flag-supplied -p / -c with the cwd and returns
// the contestRoot (parent of the contest directory, suitable as
// runner.Options.WorkDir), the contestID, and the problemID. When invoked from
// inside the contest directory itself (cwd basename == contestID), contestRoot
// drops to the parent so that joining contestRoot/contestID still yields the
// contest dir — without this, runner.Options paths would double-nest.
func resolveContest(wd, rawProblem, explicitContest string) (contestRoot, contestID, problemID string) {
	inferred, prob := parseProblemArg(rawProblem)
	cid := explicitContest
	if cid == "" {
		cid = inferred
	}
	if cid == "" {
		cid = filepath.Base(wd)
	}
	root := wd
	if filepath.Base(wd) == cid {
		root = filepath.Dir(wd)
	}
	return root, cid, prob
}

// detectLang returns the language for problemID along with the goPattern that
// was searched. The goPattern is returned regardless of outcome so callers can
// surface it in error messages without re-deriving the pattern.
func detectLang(contestRoot, contestID, problemID string) (runner.Lang, string) {
	contestDir := filepath.Join(contestRoot, contestID)
	goPattern := filepath.Join(contestDir, strings.ToUpper(problemID)+".*.go")
	matches, err := filepath.Glob(goPattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: glob pattern %q: %v\n", goPattern, err)
	}
	if len(matches) > 0 {
		return runner.LangGo, goPattern
	}

	rsFile := filepath.Join(contestRoot, contestID+".rs")
	if _, err := os.Stat(rsFile); err == nil {
		return runner.LangRust, goPattern
	}

	return "", goPattern
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
