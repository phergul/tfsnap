package snapshot

import (
	"fmt"

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
			r = Resource{
				Count: 0,
			}
		}
		r.Count++
		analysis.Resources[resource.Type] = r
	}

	return analysis, nil
}
