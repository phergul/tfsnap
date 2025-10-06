package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize TerraSnap in the current directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		workingDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		configDir := filepath.Join(workingDir, ".tfsnap")

		if _, err := os.Stat(configDir); !os.IsNotExist(err) {
			return fmt.Errorf("TerraSnap is already initialized in this directory")
		}

		return nil
	},
}
