package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/liamawhite/workstreams/pkg/types"
	workstreams "github.com/liamawhite/workstreams/pkg/workstreams"
	"github.com/spf13/cobra"
)

var switchCmd = &cobra.Command{
	Use:     "switch <name>",
	Aliases: []string{"sw"},
	Short:   "Switch to a workstream",
	Args:    cobra.ExactArgs(1),
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
		runOnLoad(dir, name)
		fmt.Fprintf(os.Stderr, "WS_CHDIR:%s\n", dir)
		return nil
	},
}

// runOnLoad executes the onLoad.sh hook for the workstream's type, if present.
func runOnLoad(dir, name string) {
	cfg, err := workstreams.ReadConfig(name)
	if err != nil || cfg.Type == "" {
		return
	}
	script := types.OnLoadScript(cfg.Type)
	if _, err := os.Stat(script); err != nil {
		return
	}
	c := exec.Command("bash", script)
	c.Dir = dir
	c.Env = append(os.Environ(),
		"WORKSTREAM_DIR="+dir,
		"WORKSTREAM_NAME="+cfg.Name,
		"WORKSTREAM_TYPE="+cfg.Type,
	)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Run() //nolint:errcheck
}
