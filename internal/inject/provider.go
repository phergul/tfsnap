package inject

import (
	"bytes"
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

var tempDir = filepath.Join(os.TempDir(), "tfsnap_tmp_module")

func RetrieveProviderSchema(cfg *config.Config, version string, localProvider bool) (*tfjson.ProviderSchema, error) {
	version = strings.TrimPrefix(version, "v")

	cacheKey := fmt.Sprintf("provider_schema_%s_%s", cfg.Provider.Name, version)
	cache := util.GetCache[tfjson.ProviderSchema](cfg.WorkingDirectory, "provider_schema")
	if !localProvider {
		schema, err := cache.Get(cacheKey)
		if err == nil && schema != nil {
			return schema, nil
		}
		if err != nil {
			log.Printf("failed to retrieve provider schema from cache: %v", err)
		}
	}

	registrySource := cfg.Provider.SourceMapping.RegistrySource
	if localProvider {
		registrySource = cfg.Provider.SourceMapping.LocalSource
	}

	if err := os.RemoveAll(tempDir); err != nil {
        log.Printf("warning: failed to clean temp dir: %v", err)
    }
    if err := os.MkdirAll(tempDir, 0o755); err != nil {
        return nil, fmt.Errorf("failed to create temp dir: %w", err)
    }

	err := createTempModule(cfg.Provider.Name, registrySource, tempDir, version)
	if err != nil {
		return nil, fmt.Errorf("Injection failed: error creating temp module: %v\n", err)
	}

	log.Println("Initialising temp module...")
	errs := terraformInit(tempDir)
	if errs != nil {
		log.Println(errs[1])
		return nil, errs[0]
	}

	log.Println("Loading provider schemas...")
	schemas, err := loadProviderSchemas(tempDir)
	if err != nil {
		fmt.Println("Injection failed: error loading provider schemas")
		log.Println(err)
		return nil, fmt.Errorf("injection failed: error loading provider schemas: %w", err)
	}

	providerSchema, key, err := resolveProviderSchemaKey(schemas, cfg)
	if err != nil {
		return nil, err
	}
	log.Printf("Resolved provider schema  key: %s\n", key)

	// schemaKey := strings.ToLower(registrySource)
	// if !localProvider {
	// 	schemaKey = "registry.terraform.io/" + schemaKey
	// }

	// log.Printf("Looking for provider schema with key: %s\n", schemaKey)
	// providerSchema, ok := schemas.Schemas[schemaKey]
	// if !ok {
	// 	return nil, fmt.Errorf("provider schema not found")
	// }

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
      source  = "%s"
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
	log.Printf("writing temp module to '%s'", tempDir)
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
		return []error{fmt.Errorf("terraform init failed; check logs for details"), fmt.Errorf("error on init in temp module: %s", string(out))}
	}
	return nil
}

func loadProviderSchemas(dir string) (*tfjson.ProviderSchemas, error) {
	cmd := exec.Command("terraform", "providers", "schema", "-json")
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("terraform error: %v, stderr: %s", err, stderr.String())
	}

	var schemas tfjson.ProviderSchemas
	if err := json.Unmarshal(stdout.Bytes(), &schemas); err != nil {
		return nil, fmt.Errorf("failed to unmarshal provider schemas: %v", err)
	}
	return &schemas, nil
}

func resolveProviderSchemaKey(schemas *tfjson.ProviderSchemas, cfg *config.Config) (*tfjson.ProviderSchema, string, error) {
    if schemas == nil || schemas.Schemas == nil || len(schemas.Schemas) == 0 {
        return nil, "", fmt.Errorf("no provider schemas returned by terraform")
    }

    regSrc := strings.ToLower(cfg.Provider.SourceMapping.RegistrySource)
    locSrc := strings.ToLower(cfg.Provider.SourceMapping.LocalSource)

    candidates := []string{}
    if regSrc != "" {
        candidates = append(candidates,
            "registry.terraform.io/"+regSrc,
            regSrc,
        )
    }
    if locSrc != "" {
        candidates = append(candidates,
            "registry.terraform.io/"+locSrc,
            locSrc,
        )
    }

    for _, k := range candidates {
        if ps, ok := schemas.Schemas[k]; ok && ps != nil {
            return ps, k, nil
        }
    }

    if len(schemas.Schemas) == 1 {
        for k, ps := range schemas.Schemas {
            if ps != nil {
                return ps, k, nil
            }
        }
    }

    nsNameSuffix := ""
    if regSrc != "" {
        parts := strings.Split(regSrc, "/")
        if len(parts) >= 2 {
            nsNameSuffix = "/" + parts[len(parts)-2] + "/" + parts[len(parts)-1]
        }
    } else if locSrc != "" {
        parts := strings.Split(locSrc, "/")
        if len(parts) >= 2 {
            nsNameSuffix = "/" + parts[len(parts)-2] + "/" + parts[len(parts)-1]
        }
    }
    if nsNameSuffix != "" {
        for k, ps := range schemas.Schemas {
            if strings.HasSuffix(strings.ToLower(k), nsNameSuffix) && ps != nil {
                return ps, k, nil
            }
        }
    }

    providerName := strings.ToLower(cfg.Provider.Name)
    if providerName != "" {
        var matchKey string
        var matchVal *tfjson.ProviderSchema
        for k, ps := range schemas.Schemas {
            if ps == nil {
                continue
            }
            if strings.HasSuffix(strings.ToLower(k), "/"+providerName) {
                if matchKey != "" {
                    matchKey = ""
                    break
                }
                matchKey = k
                matchVal = ps
            }
        }
        if matchKey != "" && matchVal != nil {
            return matchVal, matchKey, nil
        }
    }

    keys := make([]string, 0, len(schemas.Schemas))
    for k := range schemas.Schemas {
        keys = append(keys, k)
    }
    return nil, "", fmt.Errorf("provider schema not found. tried: %q; available: %v", candidates, keys)
}

func CleanupTempDir() error {
	return os.RemoveAll(tempDir)
}
