package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("version:    %s\n", Version)
		fmt.Printf("commit:     %s\n", Commit)
		fmt.Printf("build time: %s\n", BuildTime)
		return nil
	},
}
