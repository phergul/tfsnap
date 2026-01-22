package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/phergul/tfsnap/internal/autosave"
	"github.com/phergul/tfsnap/internal/config"
	"github.com/phergul/tfsnap/internal/inject"
	"github.com/phergul/tfsnap/internal/util"
	"github.com/spf13/cobra"
)

var version string
var skeleton bool
var localProvider bool
var dependency bool

var injectCmd = &cobra.Command{
	Use:    "inject <resource1>, <resource2>...",
	Short:  "Manage resources example injections",
	Args:   cobra.MinimumNArgs(1),
	PreRun: autosave.PreRun,
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.FromContext(cmd.Context())
		if cfg == nil {
			fmt.Println("configuration not found in context; run `tfsnap init` first")
			return
		}

		if !localProvider && version == "" {
			version = util.GetLatestProviderVersion(cfg)
		}
		log.Println("Using provider version:", version)

		schema, err := inject.RetrieveProviderSchema(cfg, version, localProvider)
		if err != nil {
			fmt.Printf("Injection failed: error retrieving provider schema: %v\n", err)
			return
		}
		if schema == nil {
			fmt.Println("Injection failed: provider schema is nil")
			return
		}

		for _, resourceName := range args {
			fullProviderResourceName := resourceName
			if !strings.HasPrefix(resourceName, fmt.Sprintf("%s_", cfg.Provider.Name)) {
				fullProviderResourceName = fmt.Sprintf("%s_%s", cfg.Provider.Name, resourceName)
			}

			resourceSchema, valid := inject.ValidateResource(schema, fullProviderResourceName)
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
				fmt.Println(" skeleton...")
				if err = inject.InjectSkeleton(cfg, resourceSchema, fullProviderResourceName); err != nil {
					fmt.Printf("Injection failed: %v", err)
				}
				return
			}

			if after, ok := strings.CutPrefix(resourceName, fmt.Sprintf("%s_", cfg.Provider.Name)); ok {
				resourceName = after
			}
			fmt.Println("...")
			if err = inject.InjectResource(cfg, resourceName, version, dependency); err != nil {
				fmt.Printf("Injection failed: %v\n", err)
			}
		}
	},
	PostRun: func(cmd *cobra.Command, args []string) {
		if err := inject.CleanupTempDir(); err != nil {
			log.Println("failed to delete temp dir")
		}
	},
}

func init() {
	injectCmd.Flags().StringVarP(&version, "version", "v", "", "Version of the resource")
	injectCmd.Flags().BoolVarP(&skeleton, "skeleton", "s", false, "Skeleton version of the resource")
	injectCmd.Flags().BoolVarP(&localProvider, "local", "l", false, "Use local binary (Only for skeleton)")
	injectCmd.Flags().BoolVarP(&dependency, "dependencies", "d", false, "Whether to include dependent resources")
}
