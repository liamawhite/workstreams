package cmd

import (
	"fmt"
	"os"

	workstreams "github.com/liamawhite/workstreams/pkg/workstreams"
	"github.com/spf13/cobra"
)

var switchCmd = &cobra.Command{
	Use:   "switch <name>",
	Short: "Switch to a workstream",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		ok, err := workstreams.Exists(name)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("workstream %q does not exist", name)
		}
		dir, err := workstreams.WorkstreamDir(name)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "WS_CHDIR:%s\n", dir)
		return nil
	},
}
