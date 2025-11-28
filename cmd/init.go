package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/phergul/terrasnap/internal/config"
	"github.com/spf13/cobra"
)

var configFileFlag string

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
		var cfg config.Config
		if configFileFlag != "" {
			fmt.Println("Loading configuration from:", configFileFlag)

			loaded, err := config.LoadConfig(configFileFlag)
			if err != nil {
				return fmt.Errorf("failed to read config file: %w", err)
			}

			cfg = loaded
		} else {
			reader := bufio.NewReader(os.Stdin)

			providerName := prompt(reader, "Enter provider name (e.g., aws)")
			providerDir := prompt(reader, "Enter provider directory path")
			localSource := prompt(reader, "Enter local source mapping")
			registrySource := prompt(reader, "Enter registry source mapping")

			fullProviderDir := buildProviderDir(providerDir)

			cfg = config.Config{
				WorkingDirectory: workingDir,
				Provider: config.Provider{
					Name:              providerName,
					ProviderDirectory: fullProviderDir,
					SourceMapping: config.SourceMapping{
						LocalSource:    localSource,
						RegistrySource: registrySource,
					},
				},
			}
		}

		if err := os.Mkdir(filepath.Join(configDir, "snapshots"), 0755); err != nil {
			return fmt.Errorf("failed to create snapshots directory: %w", err)
		}
		cfg.SnapshotDirectory = filepath.Join(configDir, "snapshots")

		configFile := filepath.Join(configDir, "config.yaml")
		if err := cfg.WriteConfig(configFile); err != nil {
			return err
		}

		fmt.Printf("Initialized TerraSnap in %s\n", workingDir)
		return nil
	},
}

func init() {
	initCmd.Flags().StringVarP(&configFileFlag, "config", "c", "", "Load TerraSnap config from YAML file")
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

func prompt(reader *bufio.Reader, message string) string {
	fmt.Print(message + ": ")
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}
