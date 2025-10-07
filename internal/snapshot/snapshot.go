package snapshot

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform-config-inspect/tfconfig"
)

type Provider struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	SchemaFile string `json:"schema_file"`
	Binary     string `json:"binary"`
}

type Metadata struct {
	Id        string   `json:"id"`
	CreatedAt string   `json:"created_at"`
	Provider  Provider `json:"provider"`
}

func BuildSnapshot(dir, name string) (*Metadata, error) {
	provider, err := detectProvider(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to detect provider: %w", err)
	}

	snapshotTime := getCurrentTime()
	if name == "" {
		name = fmt.Sprintf("dev-%s", snapshotTime)
	}

	return &Metadata{
		Id:        name,
		CreatedAt: snapshotTime,
		Provider:  *provider,
	}, nil
}

func detectProvider(dir string) (*Provider, error) {
	module, diag := tfconfig.LoadModule(dir)
	if diag != nil && diag.Err() != nil {
		return nil, fmt.Errorf("failed to load terraform module: %w", diag.Err())
	}

	providers := map[string]string{}
	for name, req := range module.RequiredProviders {
		providers[name] = req.VersionConstraints[0]
	}

	switch len(providers) {
	case 0:
		return nil, fmt.Errorf("no providers detected in %s", module.Path)
	case 1:
		for n, v := range providers {
			return &Provider{
				Name:    n,
				Version: v,
			}, nil
		}
	}

	return nil, fmt.Errorf("multiple providers detected in %s, please specify one explicitly", module.Path)
}

func getCurrentTime() string {
	now := time.Now()

	return now.Format("2006-01-02_15-04-05")
}
