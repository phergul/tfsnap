package cmd

import (
	"log"
	"os"

	"github.com/phergul/TerraSnap/internal/config"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tfsnap",
	Short: "A CLI tool for managing terraform developer snapshots",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.InitConfig()
		if err != nil {
			return err
		}

		cmd.SetContext(config.ToContext(cmd.Context(), &cfg))
		return nil
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(saveCmd)
}
