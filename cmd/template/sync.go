package templatecmd

import (
	"fmt"

	"github.com/liamawhite/workstreams/pkg/workspace"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync <template>",
	Short: "Sync all workspaces that use the given template",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		templateName := args[0]
		ok, err := workspace.TemplateExists(templateName)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("template %q does not exist", templateName)
		}
		workspaces, err := workspace.List()
		if err != nil {
			return err
		}
		synced := 0
		for _, ws := range workspaces {
			cfg, err := workspace.ReadConfig(ws)
			if err != nil {
				return fmt.Errorf("reading config for workspace %q: %w", ws, err)
			}
			if cfg.Template != templateName {
				continue
			}
			if err := workspace.ApplyTemplate(templateName, ws); err != nil {
				return fmt.Errorf("syncing workspace %q: %w", ws, err)
			}
			fmt.Printf("Synced workspace %q\n", ws)
			synced++
		}
		if synced == 0 {
			fmt.Printf("No workspaces use template %q\n", templateName)
		}
		return nil
	},
}
