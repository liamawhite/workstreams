package cmd

import (
	"fmt"
	"os"
	"strings"

	workstreams "github.com/liamawhite/workstreams/pkg/workstreams"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a workstream",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		dir, err := workstreams.WorkstreamDir(name)
		if err != nil {
			return err
		}
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		if err := workstreams.Delete(name); err != nil {
			return err
		}
		fmt.Printf("Removed workstream %q\n", name)
		if strings.HasPrefix(cwd, dir) {
			base, err := workstreams.BaseDir()
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "WS_CHDIR:%s\n", base)
		}
		return nil
	},
}
