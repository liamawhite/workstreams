package cmd

import (
	"fmt"
	"os"

	templatecmd "github.com/liamawhite/workspace/cmd/template"
	"github.com/spf13/cobra"
)

var Version = "dev"
var Commit = "none"
var BuildTime = "unknown"

var rootCmd = &cobra.Command{
	Use:           "workspace",
	Short:         "workspace is a CLI tool",
	SilenceUsage:  true,
	SilenceErrors: true,
	Version:       Version,
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
	rootCmd.AddCommand(templatecmd.Cmd)
}
