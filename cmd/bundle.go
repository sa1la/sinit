package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/sa1la/sinit/utils/atcoder"
	"github.com/sa1la/sinit/utils/runner"
	"github.com/spf13/cobra"
)

func init() {
	var problem string
	var contestID string
	var output string
	var copyFlag bool

	bundleCmd := &cobra.Command{
		Use:   "bundle",
		Short: "Bundle a Go solution into a single submission-ready file",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf(`unexpected positional argument %q; for output file use "-o=NAME" (space-separated form is not supported)`, args[0])
			}
			return nil
		},
		Long: `Generate a self-contained single-file Go bundle using gollect, suitable for AtCoder submission.

Only Go solutions are supported (Rust does not need bundling).

Output modes:
  (no flag)   write to stdout (suitable for piping)
  -o          write to ./submit.go
  -o=NAME     write to ./NAME
  --copy      copy bundle to system clipboard`,
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

			lang, goPattern := detectLang(contestRoot, cid, problemID)
			switch lang {
			case runner.LangRust:
				fmt.Println("Error: bundle is Go-only; Rust solutions don't need bundling")
				return
			case "":
				fmt.Printf("Error: no Go source for problem %q in contest %q (looked for %s)\n", problemID, cid, goPattern)
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
				ContestID: cid,
				ProblemID: problemID,
				WorkDir:   contestRoot,
			}

			bundlePath, err := runner.BundleGo(opts, tmpDir)
			if err != nil {
				fmt.Println("Error:", err)
				return
			}

			data, err := os.ReadFile(bundlePath)
			if err != nil {
				fmt.Println("Error reading bundle:", err)
				return
			}

			if output != "" {
				if err := os.WriteFile(output, data, 0644); err != nil {
					fmt.Println("Error writing output:", err)
					return
				}
				fmt.Printf("Bundle written to %s\n", output)
			} else if copyFlag {
				if err := copyToClipboard(string(data)); err != nil {
					fmt.Println("Error copying to clipboard:", err)
					return
				}
				fmt.Println("Bundle copied to clipboard")
			} else {
				fmt.Print(string(data))
			}
		},
	}

	bundleCmd.Flags().StringVarP(&problem, "problem", "p", "", "problem ID (e.g., c, 455c, abc455c)")
	bundleCmd.Flags().StringVarP(&contestID, "contest", "c", "", "contest ID (defaults to current directory name)")
	bundleCmd.Flags().StringVarP(&output, "output", "o", "", `output file (use "-o" alone for "submit.go", "-o=NAME" for a specific name)`)
	bundleCmd.Flags().Lookup("output").NoOptDefVal = "submit.go"
	bundleCmd.Flags().BoolVar(&copyFlag, "copy", false, "copy bundle to clipboard")

	rootCmd.AddCommand(bundleCmd)
}

func copyToClipboard(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "windows":
		cmd = exec.Command("clip")
	default:
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard", "-in")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else {
			return fmt.Errorf("no clipboard utility found (install xclip or xsel)")
		}
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
