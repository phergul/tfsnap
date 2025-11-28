package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/phergul/terrasnap/internal/config"
	"github.com/phergul/terrasnap/internal/inject"
	"github.com/spf13/cobra"
)

const tempDir = "./tmp-module"

var version string

var InjectCmd = &cobra.Command{
	Use:   "inject [<resource>]",
	Short: "Manage resources example injections",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.FromContext(cmd.Context())
		if cfg == nil {
			fmt.Println("configuration not found in context; run `tfsnap init` first")
			return
		}

		os.MkdirAll(tempDir, 0755)

		err := inject.CreateTempModule(cfg, tempDir)
		if err != nil {
			fmt.Printf("Injection failed: error creating temp module: %v\n", err)
			return
		}

		errs := inject.TerraformInit(tempDir)
		if errs != nil {
			fmt.Println(errs[0])
			log.Println(errs[1])
		}

		schemas, err := inject.LoadProviderSchemas(tempDir)
		if err != nil {
			fmt.Println("Injection failed: error loading provider schemas")
			log.Println(err)
			return
		}

		if !inject.ValidateResource(schemas, cfg.Provider.SourceMapping.RegistrySource, args[0]) {
			fmt.Printf("Resource '%s' is not valid for provider %s\n", args[0], cfg.Provider.Name)
		}
		fmt.Println("Valid Resource, injecting...")

		if version != "" && !strings.HasPrefix(version, "v") {
			version = "v" + version
		}

		resourceName := args[0]
		if after, ok := strings.CutPrefix(resourceName, fmt.Sprintf("%s_", cfg.Provider.Name)); ok {
			resourceName = after
		}
		if err = inject.InjectResource(cfg, resourceName, version); err != nil {
			fmt.Printf("Injection failed: %v\n", err)
		}
	},
	PostRun: func(cmd *cobra.Command, args []string) {
		if err := os.RemoveAll(tempDir); err != nil {
			log.Println("failed to delete temp dir")
		}
	},
}

func init() {
	InjectCmd.Flags().StringVarP(&version, "version", "v", "", "Version of the resource to inject")
}
