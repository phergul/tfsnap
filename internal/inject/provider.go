package inject

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/phergul/TerraSnap/internal/config"
)

func CreateTempModule(cfg *config.Config, dir string) error {
	temp := fmt.Sprintf(`
terraform {
  required_providers {
    %s = {
      source = "%s"
    }
  }
}
`, cfg.Provider.Name, cfg.Provider.SourceMapping.RegistrySource)

	log.Println("writing temp module to 'tmp/main.tf'")
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(temp), 0644); err != nil {
		return fmt.Errorf("failed to write temp module: %v", err)
	}
	return nil
}

func TerraformInit(dir string) []error {
	cmd := exec.Command("terraform", "init", "-no-color", "-input=false")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return []error{fmt.Errorf("terraform init failed; check logs for details"), fmt.Errorf("%s", string(out))}
	}
	return nil
}

func LoadProviderSchemas(dir string) (*tfjson.ProviderSchemas, error) {
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
