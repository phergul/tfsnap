package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/phergul/tfsnap/internal/config"
	"github.com/spf13/cobra"
)

var configFileFlag string

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize tfsnap in the current directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		workingDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		configDir := filepath.Join(workingDir, ".tfsnap")
		configFile := filepath.Join(configDir, "config.yaml")

		if _, err := os.Stat(configDir); !os.IsNotExist(err) {
			return fmt.Errorf("tfsnap is already initialized in this directory")
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

			providerDir := prompt(reader, "Enter your terraform provider directory path (e.g., ~/dev/terraform-provider-aws)")
			fullProviderDir, err := buildProviderDir(providerDir)
			if err != nil {
				return err
			}

			providerName := parseProviderName(fullProviderDir)
			if providerName == "" {
				return fmt.Errorf("failed to parse provider name from directory")
			}
			fmt.Printf("Detected provider name: %s\n", providerName)
			namespace := detectNamespace(fullProviderDir)
			if namespace != "" {
				fmt.Printf("Detected provider namespace: %s\n", namespace)
			}

			registryDefault := "registry.terraform.io/" + namespace + "/" + providerName
			fmt.Printf("Using registry source: %s\n", registryDefault)

			fmt.Print("Do you have a custom source for local provider development? (y/N): ")
			choice, _ := reader.ReadString('\n')
			choice = strings.ToLower(strings.TrimSpace(choice))

			localSource := ""
			if choice != "y" && choice != "yes" {
				localSource = prompt(reader, fmt.Sprintf("Enter local source"))
			} else {
				localSource = ""
			}

			cfg = config.Config{
				ConfigPath:       configFile,
				WorkingDirectory: workingDir,
				Provider: config.Provider{
					Name:              providerName,
					ProviderDirectory: fullProviderDir,
					SourceMapping: config.SourceMapping{
						LocalSource:    localSource,
						RegistrySource: registryDefault,
					},
				},
			}
		}

		if err := os.Mkdir(filepath.Join(configDir, "snapshots"), 0755); err != nil {
			return fmt.Errorf("failed to create snapshots directory: %w", err)
		}
		cfg.SnapshotDirectory = filepath.Join(configDir, "snapshots")

		if err := cfg.WriteConfig(); err != nil {
			return err
		}

		fmt.Printf("Initialized tfsnap in %s\n", workingDir)
		return nil
	},
}

func init() {
	initCmd.Flags().StringVarP(&configFileFlag, "config", "c", "", "Load tfsnap config from YAML file")
}

func buildProviderDir(dir string) (string, error) {
	if dir == "" {
		return "", fmt.Errorf("provider directory is required")
	}
	if filepath.IsAbs(dir) {
		return dir, nil
	} else if strings.HasPrefix(dir, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		return filepath.Join(homeDir, dir[1:]), nil
	}

	return "", fmt.Errorf("provider directory must be an absolute path or start with ~")
}

func prompt(reader *bufio.Reader, message string) string {
	fmt.Print(message + ": ")
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func parseProviderName(providerDir string) string {
	base := filepath.Base(providerDir)
	parts := strings.Split(base, "-")
	if len(parts) >= 3 {
		return parts[2]
	}
	return base
}

func detectNamespace(providerDir string) string {
	if providerDir == "" {
		return ""
	}

	goModPath := filepath.Join(providerDir, "go.mod")
	data, err := os.ReadFile(goModPath)
	if err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "module ") {
				module := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "module "))
				parts := strings.Split(module, "/")
				if len(parts) >= 2 {
					return parts[1]
				}
			}
		}
	}

	return ""
}
