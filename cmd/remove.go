package cmd

import (
	"fmt"
	"os"
	"strings"

	workstreams "github.com/liamawhite/workstreams/pkg/workstreams"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:     "remove [name]",
	Aliases: []string{"rm"},
	Short:   "Remove a workstream (defaults to the current workstream)",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		// No arg: remove whichever workstream cwd is inside, then exit the shell.
		if len(args) == 0 {
			dirName, cfg, err := workstreams.ForDir(cwd)
			if err != nil {
				return err
			}
			if err := workstreams.Delete(dirName); err != nil {
				return err
			}
			fmt.Printf("Removed workstream %q\n", cfg.Name)
			fmt.Fprintln(os.Stderr, "WS_EXIT")
			return nil
		}

		// Named removal: existing behaviour.
		name := args[0]
		dir, err := workstreams.WorkstreamDir(name)
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
