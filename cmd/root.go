package cmd

import (
	"fmt"
	"os"
	"os/exec"

	typecmd "github.com/liamawhite/workstreams/cmd/types"
	"github.com/liamawhite/workstreams/internal/tui"
	"github.com/liamawhite/workstreams/pkg/types"
	workstreams "github.com/liamawhite/workstreams/pkg/workstreams"
	"github.com/spf13/cobra"
)

var Version = "dev"
var Commit = "none"
var BuildTime = "unknown"

var rootCmd = &cobra.Command{
	Use:           "workstreams",
	Short:         "workstreams is a CLI tool",
	SilenceUsage:  true,
	SilenceErrors: true,
	Version:       Version,
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := tui.Run()
		if err != nil {
			return err
		}

		// User confirmed creation: clear screen then stream onInit.sh to terminal.
		if result.CreateName != "" {
			fmt.Print("\033[H\033[2J")
			if _, err := workstreams.Create(result.CreateName, result.CreateType); err != nil {
				return err
			}
			result.Selected = workstreams.ToDirName(result.CreateName)
		}

		if result.Selected == "" {
			return nil
		}

		dir, err := workstreams.WorkstreamDir(result.Selected)
		if err != nil {
			return err
		}
		cfg, err := workstreams.ReadConfig(result.Selected)
		if err == nil && cfg.Type != "" {
			script := types.OnLoadScript(cfg.Type)
			if _, statErr := os.Stat(script); statErr == nil {
				c := exec.Command("bash", script)
				c.Dir = dir
				c.Env = append(os.Environ(),
					"WORKSTREAM_DIR="+dir,
					"WORKSTREAM_NAME="+cfg.Name,
					"WORKSTREAM_TYPE="+cfg.Type,
				)
				c.Stdout = os.Stdout
				c.Stderr = os.Stderr
				c.Run() //nolint:errcheck
			}
		}
		fmt.Fprintf(os.Stderr, "WS_CHDIR:%s\n", dir)
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "enable verbose output")
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(switchCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(typecmd.Cmd)
}
