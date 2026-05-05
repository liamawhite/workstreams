package cmd

import (
	"fmt"
	"os"

	"github.com/liamawhite/workspace/pkg/workspace"
	"github.com/spf13/cobra"
)

var addTemplate string

var addCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Create a new workspace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := workspace.Create(args[0], addTemplate)
		if err != nil {
			return err
		}
		fmt.Printf("Created workspace %q\n", args[0])
		fmt.Fprintf(os.Stderr, "WS_CHDIR:%s\n", dir)
		return nil
	},
}

func init() {
	addCmd.Flags().StringVar(&addTemplate, "template", "", "template name to apply to the new workspace")
}
