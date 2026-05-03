package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sa1la/sinit/utils/atcoder"
	"github.com/sa1la/sinit/utils/runner"
	"github.com/spf13/cobra"
)

func init() {
	var problem string
	var contestID string
	var output string

	bundleCmd := &cobra.Command{
		Use:   "bundle",
		Short: "Bundle a Go solution into a single submission-ready file",
		Long:  "Generate a self-contained single-file Go bundle using gollect, suitable for submission to AtCoder.",
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

			inferred, prob := parseProblemArg(problem)
			problem = prob
			if contestID == "" {
				contestID = inferred
			}
			if contestID == "" {
				contestID = filepath.Base(wd)
			}

			lang := detectLang(wd, filepath.Join(wd, contestID), contestID, problem)
			if lang != runner.LangGo {
				fmt.Printf("Error: bundle only supports Go solutions (detected: %s)\n", lang)
				return
			}

			tmpDir, err := os.MkdirTemp("", "sinit-bundle-*")
			if err != nil {
				fmt.Println("Error:", err)
				return
			}
			defer os.RemoveAll(tmpDir)

			opts := runner.Options{
				Lang:      runner.LangGo,
				ContestID: contestID,
				ProblemID: problem,
				WorkDir:   wd,
			}

			gollectBin, err := exec.LookPath("gollect")
			if err != nil {
				fmt.Println("Error: gollect not found in PATH")
				return
			}

			bundlePath, err := runner.BundleGo(opts, tmpDir, gollectBin)
			if err != nil {
				fmt.Println("Error:", err)
				return
			}

			if output != "" {
				if err := os.Rename(bundlePath, output); err != nil {
					fmt.Println("Error writing output:", err)
					return
				}
				fmt.Printf("Bundle written to %s\n", output)
			} else {
				data, err := os.ReadFile(bundlePath)
				if err != nil {
					fmt.Println("Error reading bundle:", err)
					return
				}
				fmt.Print(string(data))
			}
		},
	}

	bundleCmd.Flags().StringVarP(&problem, "problem", "p", "", "problem ID (e.g., c, 455c, abc455c)")
	bundleCmd.Flags().StringVarP(&contestID, "contest", "c", "", "contest ID (defaults to current directory name)")
	bundleCmd.Flags().StringVarP(&output, "output", "o", "", "output file (default: stdout)")

	rootCmd.AddCommand(bundleCmd)
}
