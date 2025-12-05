package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/phergul/tfsnap/internal/config"
	"github.com/phergul/tfsnap/internal/util"
	"github.com/spf13/cobra"
)

var local bool

var versionCmd = &cobra.Command{
	Use:   "version <version>",
	Short: "Change the version of the current terraform config",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.FromContext(cmd.Context())
		if cfg == nil {
			fmt.Println("configuration not found in context; run `tfsnap init` first")
			return
		}
		version := args[0]

		tfFile := filepath.Join(cfg.WorkingDirectory, "main.tf")
		data, err := os.ReadFile(tfFile)
		if err != nil {
			fmt.Println("unable to open main.tf file")
			return
		}

		if !local {
			if version == "latest" {
				version = strings.TrimPrefix(util.GetLatestProviderVersion(cfg), "v")
			} else {
				versions, err := util.GetAvailableProviderVersions(cfg.Provider.SourceMapping.RegistrySource)
				if err != nil {
					fmt.Println("failed to get available provider versions")
					return
				}
				if !slices.ContainsFunc(versions, func(v string) bool {
					return v == "v"+version
				}) {
					fmt.Println("specified version is not available")
					return
				}
			}
		}

		log.Println("Changing provider to version", version)
		fileContent := string(data)
		newContent := fileContent

		newVersion := fmt.Sprintf(`version = "%s"`, version)

		targetSource := cfg.Provider.SourceMapping.RegistrySource
		if local {
			if cfg.Provider.SourceMapping.LocalSource == "" {
				fmt.Println("LocalSource must be set in tfsnap config to use local version")
				return
			}
			targetSource = cfg.Provider.SourceMapping.LocalSource
		}
		newSource := fmt.Sprintf(`source = "%s"`, targetSource)

		versionRe := regexp.MustCompile(`version\s*=\s*"[0-9]+\.[0-9]+\.[0-9]+"`)
		if versionRe.MatchString(newContent) {
			newContent = versionRe.ReplaceAllString(newContent, newVersion)
		} else {
			log.Println("No existing version constraint found, adding...")
			newSource = fmt.Sprintf("%s\n\t\t\t%s", newSource, newVersion)
		}

		sourceRe := regexp.MustCompile(`source\s*=\s*"[^"]+"`)
		if !sourceRe.MatchString(newContent) {
			fmt.Println("No existing source found")
			return
		}

		newContent = sourceRe.ReplaceAllString(newContent, newSource)

		err = os.WriteFile(tfFile, []byte(newContent), 0644)
		if err != nil {
			fmt.Println("failed to write to main.tf file")
			return
		}

		fmt.Println("Provider version updated to:", version)
	},
}

func init() {
	versionCmd.Flags().BoolVarP(&local, "local", "l", false, "Whether this version is the local development version")
}
