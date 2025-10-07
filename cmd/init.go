package cmd

import (
	"fmt"
	"log"
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

		if err := os.Mkdir(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		fullProviderDir := buildProviderDir(providerDirectory)

		config := config.Config{
			WorkingDirectory: workingDir,
			Provider: config.Provider{
				Name:              providerName,
				LocalBuildCommand: localBuildCommand,
				ProviderDirectory: fullProviderDir,
			},
		}
		log.Printf("init config %+v", config)

		configFile := filepath.Join(configDir, "config.yaml")
		if err := config.WriteConfig(configFile); err != nil {
			return err
		}

		if err := os.Mkdir(filepath.Join(configDir, "snapshots"), 0755); err != nil {
			return fmt.Errorf("failed to create snapshots directory: %w", err)
		}
		config.SnapshotDirectory = filepath.Join(configDir, "snapshots")

		fmt.Printf("Initialized TerraSnap in %s\n", configDir)
		return nil
	},
}

func init() {
	initCmd.Flags().StringVarP(&providerName, "provider", "p", "", "Terraform provider name (required)")
	initCmd.Flags().StringVarP(&localBuildCommand, "build-command", "b", "", "Local build command for the provider (required)")
	initCmd.Flags().StringVarP(&providerDirectory, "provider-dir", "d", "", "Provider directory (required)")
	initCmd.MarkFlagRequired("provider")
}

func buildProviderDir(dir string) string {
	if dir == "" {
		return ""
	}
	if filepath.IsAbs(dir) {
		return dir
	}
	wd, err := os.Getwd()
	if err != nil {
		return dir
	}
	return filepath.Join(wd, dir)
}
