package typecmd

import (
	"fmt"

	"github.com/liamawhite/workstreams/pkg/types"
	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:   "rm <name>",
	Short: "Remove a workstream type",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := types.Delete(args[0]); err != nil {
			return err
		}
		fmt.Printf("Removed type %q\n", args[0])
		return nil
	},
}
