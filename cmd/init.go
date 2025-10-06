package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/phergul/TerraSnap/internal/config"
	"github.com/spf13/cobra"
)

var (
	providerName      string
	localBuildCommand string
	providerDirectory string
)

func init() {
	initCmd.Flags().StringVarP(&providerName, "provider", "p", "", "Terraform provider name (required)")
	initCmd.Flags().StringVarP(&localBuildCommand, "build-command", "b", "", "Local build command for the provider (required)")
	initCmd.Flags().StringVarP(&providerDirectory, "provider-dir", "d", "", "Provider directory (required)")
	initCmd.MarkFlagRequired("provider")
	initCmd.MarkFlagRequired("build-command")
}

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
			return fmt.Errorf("TerraSnap is already initialized in this directory\n")
		}

		if err := os.Mkdir(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		config := config.Config{
			Provider: config.Provider{
				Name:              providerName,
				LocalBuildCommand: localBuildCommand,
				ProviderDirectory: providerDirectory,
			},
		}

		configFile := filepath.Join(configDir, "config.yaml")
		if err := config.WriteConfig(configFile); err != nil {
			return err
		}

		fmt.Printf("Initialized TerraSnap in %s\n", configDir)
		return nil
	},
}