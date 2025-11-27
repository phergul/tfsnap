package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"github.com/phergul/TerraSnap/internal/config"
	"github.com/spf13/cobra"
)

var local bool

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Change the version of the current terraform config",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.FromContext(cmd.Context())
		if cfg == nil {
			fmt.Println("configuration not found in context; run `tfsnap init` first")
			return
		}

		tfFile := filepath.Join(cfg.WorkingDirectory, "main.tf")
		data, err := os.ReadFile(tfFile)
		if err != nil {
			fmt.Println("unable to open main.tf file")
			return
		}

		log.Println("Changing provider to version", args[0])
		fileContent := string(data)
		versionRe := regexp.MustCompile(`version\s*=\s*"[0-9]+\.[0-9]+\.[0-9]+"`)
		newContent := versionRe.ReplaceAllString(fileContent, fmt.Sprintf(`version = "%s"`, args[0]))

		sourceRe := regexp.MustCompile(`source\s*=\s*"[^"]+"`)
		if local {
			if cfg.Provider.SourceMapping.LocalSource == "" {
				fmt.Println("LocalSource must be set in tfsnap config to use local version")
				return
			}
			newContent = sourceRe.ReplaceAllString(newContent, fmt.Sprintf(`source = "%s"`, cfg.Provider.SourceMapping.LocalSource))
		} else {
			newContent = sourceRe.ReplaceAllString(newContent, fmt.Sprintf(`source = "%s"`, cfg.Provider.SourceMapping.RegistrySource))
		}

		err = os.WriteFile(tfFile, []byte(newContent), 0644)
		if err != nil {
			fmt.Println("failed to write to main.tf file")
			return
		}

		fmt.Println("Provider version updated to:", args[0])
	},
}

func init() {
	versionCmd.Flags().BoolVarP(&local, "local", "l", false, "Whether this version is the local development version")
}
