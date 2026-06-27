package typecmd

import (
	"fmt"

	"github.com/liamawhite/workstreams/pkg/types"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Create a new workstream type",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := types.Create(args[0])
		if err != nil {
			return err
		}
		fmt.Printf("Created type %q at %s\n", args[0], dir)
		return nil
	},
}
