package typecmd

import "github.com/spf13/cobra"

var Cmd = &cobra.Command{
	Use:     "types",
	Aliases: []string{"type", "typ", "t"},
	Short:   "Manage workstream types",
}

func init() {
	Cmd.AddCommand(addCmd)
	Cmd.AddCommand(rmCmd)
}
