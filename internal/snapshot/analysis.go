package snapshot

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform-config-inspect/tfconfig"
)

func AnalyseTFConfig(config string) (*ConfigAnalysis, error) {
	module, err := tfconfig.LoadModule(config)
	if err != nil {
		return nil, fmt.Errorf("failed to load Terraform module: %w", err)
	}

	analysis := &ConfigAnalysis{
		Resources:  make(map[string]Resource),
		TotalCount: len(module.ManagedResources),
	}

	for _, resource := range module.ManagedResources {
		r, exists := analysis.Resources[resource.Type]
		if !exists {
			log.Println("New resource for:", resource.Type)
			r = Resource{
				Type:  resource.Type,
				Count: 0,
			}
		}
		log.Printf("[%s] incrementing count\n", resource.Type)
		r.Count++
		analysis.Resources[resource.Type] = r
	}

	return analysis, nil
}
