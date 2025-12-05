package inject

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/phergul/tfsnap/internal/config"
	"github.com/phergul/tfsnap/internal/util"
)

const tempDir = "./tmp-module"

func RetrieveProviderSchema(cfg *config.Config, version string, localProvider bool) (*tfjson.ProviderSchema, error) {
	version = strings.TrimPrefix(version, "v")

	cacheKey := fmt.Sprintf("provider_schema_%s_%s", cfg.Provider.Name, version)
	cache := util.GetCache[tfjson.ProviderSchema](cfg.WorkingDirectory, "provider_schema")
	if !localProvider {
		schema, err := cache.Get(cacheKey)
		if err == nil {
			return schema, nil
		}
		log.Println(err)
	}

	registrySource := cfg.Provider.SourceMapping.RegistrySource
	if localProvider {
		registrySource = cfg.Provider.SourceMapping.LocalSource
	}

	os.MkdirAll(tempDir, 0755)

	err := createTempModule(cfg.Provider.Name, registrySource, tempDir, version)
	if err != nil {
		return nil, fmt.Errorf("Injection failed: error creating temp module: %v\n", err)
	}

	log.Println("Initialising temp module...")
	errs := terraformInit(tempDir)
	if errs != nil {
		fmt.Println(errs[0])
		log.Println(errs[1])
		return nil, nil
	}

	log.Println("Loading provider schemas...")
	schemas, err := loadProviderSchemas(tempDir)
	if err != nil {
		fmt.Println("Injection failed: error loading provider schemas")
		log.Println(err)
		return nil, nil
	}

	schemaKey := registrySource
	if !localProvider {
		schemaKey = "registry.terraform.io/" + registrySource
	}

	providerSchema, ok := schemas.Schemas[schemaKey]
	if !ok {
		return nil, fmt.Errorf("provider schema not found")
	}

	if !localProvider {
		if err := cache.Set(cacheKey, *providerSchema); err != nil {
			log.Printf("failed to cache provider schema: %v", err)
		}
	}

	return providerSchema, nil
}

func createTempModule(name, source, dir, version string) error {
	var temp string
	if version != "" {
		temp = fmt.Sprintf(`
		terraform {
  required_providers {
    %s = {
      source = "%s"
	  version = "%s"
    }
  }
}
`, name, source, version)
	} else {
		temp = fmt.Sprintf(`
terraform {
  required_providers {
    %s = {
      source = "%s"
    }
  }
}
`, name, source)
	}
	log.Println("writing temp module to 'tmp/main.tf'")
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(temp), 0644); err != nil {
		return fmt.Errorf("failed to write temp module: %v", err)
	}
	return nil
}

func terraformInit(dir string) []error {
	cmd := exec.Command("terraform", "init", "-no-color", "-input=false", "-backend=false")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return []error{fmt.Errorf("terraform init failed; check logs for details"), fmt.Errorf("%s", string(out))}
	}
	return nil
}

func loadProviderSchemas(dir string) (*tfjson.ProviderSchemas, error) {
	cmd := exec.Command("terraform", "providers", "schema", "-json")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to load provider schemas: %v", err)
	}

	var schemas tfjson.ProviderSchemas
	if err := json.Unmarshal(out, &schemas); err != nil {
		return nil, fmt.Errorf("failed to unmarshal provider schemas: %v", err)
	}
	return &schemas, nil
}

func CleanupTempDir() error {
	return os.RemoveAll(tempDir)
}
