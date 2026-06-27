package cmd

import (
	"fmt"
	"os"

	"github.com/liamawhite/workstreams/pkg/types"
	workstreams "github.com/liamawhite/workstreams/pkg/workstreams"
	"github.com/spf13/cobra"
)

var addType string

var addCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Create a new workstream",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := workstreams.Create(args[0], addType)
		if err != nil {
			return err
		}
		fmt.Printf("Created workstream %q\n", args[0])
		fmt.Fprintf(os.Stderr, "WS_CHDIR:%s\n", dir)
		return nil
	},
}

func init() {
	addCmd.Flags().StringVar(&addType, "type", "", "workstream type (must match an existing type dir)")
	//nolint:errcheck
	addCmd.RegisterFlagCompletionFunc("type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		names, err := types.List()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	})
}
