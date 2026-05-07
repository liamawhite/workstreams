package cmd

import (
	"fmt"
	"os"

	workstreams "github.com/liamawhite/workstreams/pkg/workstreams"
	"github.com/spf13/cobra"
)

var addTemplate string

var addCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Create a new workstream",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := workstreams.Create(args[0], addTemplate)
		if err != nil {
			return err
		}
		fmt.Printf("Created workstream %q\n", args[0])
		fmt.Fprintf(os.Stderr, "WS_CHDIR:%s\n", dir)
		return nil
	},
}

func init() {
	addCmd.Flags().StringVar(&addTemplate, "template", "", "template name to apply to the new workstream")
}
