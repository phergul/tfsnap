package cmd

import (
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

		cfg, err := config.InitConfig()
		if err != nil {
			return err
		}

		cmd.SetContext(config.ToContext(cmd.Context(), &cfg))
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
