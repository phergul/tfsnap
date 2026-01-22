package cmd

import (
	"fmt"

	"github.com/phergul/tfsnap/internal/config"
	"github.com/phergul/tfsnap/internal/template"
	"github.com/spf13/cobra"
)

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Manage resource templates",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.FromContext(cmd.Context())
		if cfg == nil {
			fmt.Println("configuration not found in context; run `tfsnap init` first")
			return nil
		}

		return template.Run(cfg)
	},
}

func init() {
	templateCmd.AddCommand(newTemplateSaveCmd())
}

func newTemplateSaveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "save <template-name>",
		Short: "Save a resource from main.tf as a template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())
			if cfg == nil {
				fmt.Println("configuration not found in context; run `tfsnap init` first")
				return nil
			}

			return template.RunSave(cfg, args[0])
		},
	}
	return cmd
}
