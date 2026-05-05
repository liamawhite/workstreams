package templatecmd

import "github.com/spf13/cobra"

var Cmd = &cobra.Command{
	Use:   "template",
	Short: "Manage workspace templates",
}

func init() {
	Cmd.AddCommand(addCmd)
	Cmd.AddCommand(rmCmd)
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(syncCmd)
}
