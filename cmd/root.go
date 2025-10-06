package cmd

import (
	"fmt"
	"os"

	"github.com/phergul/TerraSnap/internal/config"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tfsnap",
	Short: "A CLI tool for managing terraform developer snapshots",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == "init" {
			return nil
		}

		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w\ntry running 'tfsnap init' first", err)
		}

		cmd.SetContext(cfg.ToContext(cmd.Context()))	
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(saveCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
