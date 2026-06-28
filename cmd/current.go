package cmd

import (
	"fmt"
	"os"

	workstreams "github.com/liamawhite/workstreams/pkg/workstreams"
	"github.com/spf13/cobra"
)

var currentCmd = &cobra.Command{
	Use:   "current",
	Short: "Print the name of the workstream for the current directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}
		cfg, err := workstreams.ForDir(cwd)
		if err != nil {
			return err
		}
		fmt.Println(cfg.Name)
		return nil
	},
}
