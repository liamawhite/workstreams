package templatecmd

import (
	"fmt"

	workstreams "github.com/liamawhite/workstreams/pkg/workstreams"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync <template>",
	Short: "Sync all workstreams that use the given template",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		templateName := args[0]
		ok, err := workstreams.TemplateExists(templateName)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("template %q does not exist", templateName)
		}
		all, err := workstreams.List()
		if err != nil {
			return err
		}
		synced := 0
		for _, ws := range all {
			cfg, err := workstreams.ReadConfig(ws)
			if err != nil {
				return fmt.Errorf("reading config for workstream %q: %w", ws, err)
			}
			if cfg.Template != templateName {
				continue
			}
			if err := workstreams.ApplyTemplate(templateName, ws); err != nil {
				return fmt.Errorf("syncing workstream %q: %w", ws, err)
			}
			fmt.Printf("Synced workstream %q\n", ws)
			synced++
		}
		if synced == 0 {
			fmt.Printf("No workstreams use template %q\n", templateName)
		}
		return nil
	},
}
