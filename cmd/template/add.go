package templatecmd

import (
	"fmt"

	workstreams "github.com/liamawhite/workstreams/pkg/workstreams"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Create a new template",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := workstreams.CreateTemplate(args[0]); err != nil {
			return err
		}
		dir, err := workstreams.TemplateDir(args[0])
		if err != nil {
			return err
		}
		fmt.Printf("Created template %q at %s\n", args[0], dir)
		return nil
	},
}
