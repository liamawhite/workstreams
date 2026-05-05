package cmd

import (
	"fmt"
	"os"

	"github.com/liamawhite/workspace/pkg/workspace"
	"github.com/spf13/cobra"
)

var switchCmd = &cobra.Command{
	Use:   "switch <name>",
	Short: "Switch to a workspace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		ok, err := workspace.Exists(name)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("workspace %q does not exist", name)
		}
		dir, err := workspace.WorkspaceDir(name)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "WS_CHDIR:%s\n", dir)
		return nil
	},
}
