package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/phergul/tfsnap/internal/config"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tfsnap",
	Short: "A CLI tool for managing terraform developer snapshots",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == "init" {
			return nil
		}

		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		file, err := os.OpenFile(filepath.Join(wd, ".tfsnap/tfsnap.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return fmt.Errorf("no .tfsnap directory found in current working directory; run `tfsnap init` first")
		}
		log.SetOutput(file)
		log.SetFlags(log.Ldate | log.Ltime)

		fullCmd := strings.ToUpper(cmd.Name())
		currParent := cmd.Parent()
		for {
			if currParent == cmd.Root() {
				break
			}
			fullCmd = strings.ToUpper(currParent.Name()) + " " + fullCmd
			currParent = currParent.Parent()
		}
		log.Printf("[%s] Start - args: {%s}", fullCmd, strings.Join(args, ", "))

		cfg, err := config.LoadConfig("")
		if err != nil {
			return fmt.Errorf("failed to load config: %w\ntry running 'tfsnap init' first", err)
		}

		cmd.SetContext(config.ToContext(cmd.Context(), &cfg))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(snapshotCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(injectCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
