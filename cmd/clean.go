package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/phergul/tfsnap/internal/config"
	"github.com/spf13/cobra"
)

var exclude []string

var filesToRemove = []string{
	".terraform.lock.hcl",
	"terraform.tfstate",
	"terraform.tfstate.backup",
}

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean up terraform files",
	Long:  "Remove local terraform files such as .terraform.lock.hcl and terraform.tfstate files.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.FromContext(cmd.Context())
		if cfg == nil {
			fmt.Println("configuration not found in context; run `tfsnap init` first")
			return
		}

		filteredFiles := []string{}

		for _, file := range filesToRemove {
			if slices.Contains(exclude, file) {
				continue
			}
			filteredFiles = append(filteredFiles, file)
		}

		for _, file := range filteredFiles {
			err := removeFileIfExists(cfg, file)
			if err != nil {
				fmt.Printf("error removing file %q: %v\n", file, err)
			}
		}

		fmt.Println("Cleanup completed")
	},
}

func init() {
	cleanCmd.Flags().StringArrayVarP(&exclude, "exclude", "e", []string{}, "Files to exclude from cleanup")
}

func removeFileIfExists(cfg *config.Config, filename string) error {
	filePath := filepath.Join(cfg.WorkingDirectory, filename)
	return os.RemoveAll(filePath)
}
