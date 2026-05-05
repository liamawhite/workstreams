package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/liamawhite/workspace/pkg/workspace"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a workspace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		dir, err := workspace.WorkspaceDir(name)
		if err != nil {
			return err
		}
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		if err := workspace.Delete(name); err != nil {
			return err
		}
		fmt.Printf("Removed workspace %q\n", name)
		if strings.HasPrefix(cwd, dir) {
			base, err := workspace.BaseDir()
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "WS_CHDIR:%s\n", base)
		}
		return nil
	},
}
