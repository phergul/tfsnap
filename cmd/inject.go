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
var skeleton bool
var localProvider bool
var dependency bool

var InjectCmd = &cobra.Command{
	Use:   "inject <resource1>, <resource2>...",
	Short: "Manage resources example injections",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.FromContext(cmd.Context())
		if cfg == nil {
			fmt.Println("configuration not found in context; run `tfsnap init` first")
			return
		}

		os.MkdirAll(tempDir, 0755)

		registrySource := cfg.Provider.SourceMapping.RegistrySource
		if localProvider {
			registrySource = cfg.Provider.SourceMapping.LocalSource
		}

		err := inject.CreateTempModule(cfg.Provider.Name, registrySource, tempDir, version)
		if err != nil {
			fmt.Printf("Injection failed: error creating temp module: %v\n", err)
			return
		}

		log.Println("Initialising temp module...")
		errs := inject.TerraformInit(tempDir)
		if errs != nil {
			fmt.Println(errs[0])
			log.Println(errs[1])
			return
		}

		log.Println("Loading provider schemas...")
		schemas, err := inject.LoadProviderSchemas(tempDir)
		if err != nil {
			fmt.Println("Injection failed: error loading provider schemas")
			log.Println(err)
			return
		}

		schemaKey := registrySource
		if !localProvider {
			schemaKey = "registry.terraform.io/" + registrySource
		}

		for _, resourceName := range args {
			fullProviderResourceName := resourceName
			if !strings.HasPrefix(resourceName, fmt.Sprintf("%s_", cfg.Provider.Name)) {
				fullProviderResourceName = fmt.Sprintf("%s_%s", cfg.Provider.Name, resourceName)
			}

			schema, valid := inject.ValidateResource(schemas, schemaKey, fullProviderResourceName)
			if !valid {
				fmt.Printf("Resource '%s' is not valid for provider %s@", args[0], cfg.Provider.Name)
				if version != "" {
					fmt.Println(version)
				} else {
					fmt.Println("latest")
				}
				return
			}
			fmt.Printf("Valid resource [%s]. Injecting", resourceName)

			if version != "" && !strings.HasPrefix(version, "v") {
				version = "v" + version
			}

			if skeleton {
				fmt.Println("skeleton...")
				if err = inject.InjectSkeleton(cfg, schema, fullProviderResourceName); err != nil {
					fmt.Printf("Injection failed: %v", err)
				}
				return
			}

			if after, ok := strings.CutPrefix(resourceName, fmt.Sprintf("%s_", cfg.Provider.Name)); ok {
				resourceName = after
			}
			fmt.Println("...")
			if err = inject.InjectResource(cfg, resourceName, version); err != nil {
				fmt.Printf("Injection failed: %v\n", err)
			}
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
	InjectCmd.Flags().BoolVarP(&skeleton, "skeleton", "s", false, "Inject a skeleton version of the resource")
	InjectCmd.Flags().BoolVarP(&localProvider, "local", "l", false, "Use local binary (currently only used for skeleton")
	InjectCmd.Flags().BoolVarP(&dependency, "dependencies", "d", false, "Whether to include dependent resources in the injection")
}
