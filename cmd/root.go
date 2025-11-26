package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

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

		file, err := os.OpenFile(filepath.Join(cfg.WorkingDirectory, ".tfsnap/tfsnap.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return fmt.Errorf("failed to create log file: %w", err)
		}

		log.SetOutput(file)
		log.SetFlags(log.Ldate | log.Ltime)

		cmd.SetContext(config.ToContext(cmd.Context(), &cfg))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(snapshotCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalln(err)
	}
}
