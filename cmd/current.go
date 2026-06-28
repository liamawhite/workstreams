package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	workstreams "github.com/liamawhite/workstreams/pkg/workstreams"
	"github.com/spf13/cobra"
)

var currentCmd = &cobra.Command{
	Use:   "current",
	Short: "Print the name of the workstream for the current directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}
		base, err := workstreams.BaseDir()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(base, cwd)
		if err != nil || strings.HasPrefix(rel, "..") || rel == "." {
			return errors.New("not inside a workstream directory")
		}
		dirName := strings.SplitN(rel, string(os.PathSeparator), 2)[0]
		cfg, err := workstreams.ReadConfig(dirName)
		if err != nil {
			return err
		}
		fmt.Println(cfg.Name)
		return nil
	},
}
