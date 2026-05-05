package templatecmd

import (
	"fmt"

	"github.com/liamawhite/workspace/pkg/workspace"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Create a new template",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := workspace.CreateTemplate(args[0]); err != nil {
			return err
		}
		dir, err := workspace.TemplateDir(args[0])
		if err != nil {
			return err
		}
		fmt.Printf("Created template %q at %s\n", args[0], dir)
		return nil
	},
}
