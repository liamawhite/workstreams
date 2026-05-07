package templatecmd

import (
	"fmt"

	workstreams "github.com/liamawhite/workstreams/pkg/workstreams"
	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:   "rm <name>",
	Short: "Delete a template",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := workstreams.DeleteTemplate(args[0]); err != nil {
			return err
		}
		fmt.Printf("Deleted template %q\n", args[0])
		return nil
	},
}
