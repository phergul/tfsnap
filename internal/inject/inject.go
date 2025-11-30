package inject

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/manifoldco/promptui"
	"github.com/phergul/terrasnap/internal/config"
	"github.com/phergul/terrasnap/internal/util"
	"golang.org/x/mod/semver"
)

type ProviderVersion struct {
	Version string `json:"version"`
}

type VersionResponse struct {
	Versions []ProviderVersion `json:"versions"`
}

func ValidateResource(schemas *tfjson.ProviderSchemas, registrySource, input string) (*tfjson.Schema, bool) {
	log.Println("schemaKey is:", registrySource)

	providerSchema, ok := schemas.Schemas[registrySource]
	if !ok {
		return nil, false
	}

	schema, ok := providerSchema.ResourceSchemas[input]
	return schema, ok
}

func InjectResource(cfg *config.Config, resourceType, version string) error {
	tfPath := filepath.Join(cfg.WorkingDirectory, "main.tf")

	resource, err := getResourceExample(cfg, resourceType, version)
	if err != nil {
		log.Println(err)
		return fmt.Errorf("failed to inject resource. Check logs for details.")
	}

	return writeResourceToFile(tfPath, resource)
}

func writeResourceToFile(path, resource string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	prefix := ""
	if len(content) >= 2 && !(content[len(content)-2] == '\n' && content[len(content)-1] == '\n') {
		prefix = "\n"
	}

	_, err = file.WriteString(prefix + resource + "\n\n")
	return err
}

func getResourceExample(cfg *config.Config, resourceType, version string) (string, error) {
	versions, err := getAvailableProviderVersions(cfg.Provider.SourceMapping.RegistrySource)
	if err != nil {
		log.Printf("failed to get provider versions for %s: %v", strings.Split(cfg.Provider.SourceMapping.RegistrySource, "/")[:1], err)
		return "", err
	}

	var providerVersion string
	if version == "" {
		providerVersion = versions[0] //latest
	} else {
		if !slices.Contains(versions, version) {
			return "", fmt.Errorf("provided version %s does not exist for provider", version)
		}
		providerVersion = version
	}

	examplesClient := NewExampleClient(cfg)

	examples, err := examplesClient.findGithubExamples(providerVersion, resourceType)
	if err != nil {
		return "", fmt.Errorf("failed to get examples from github repo (%s): %w", examplesClient.providerMetadata.Source, err)
	}

	if len(*examples) > 1 {
		fmt.Printf("Multiple %s resources found\n", resourceType)
		prompt := promptui.Select{
			Label: fmt.Sprintf("Select %s example to inject", resourceType),
			Items: *examples,
			Templates: &promptui.SelectTemplates{
				Label:    "{{ . }}:",
				Active:   "> {{ .Name | underline }}",
				Inactive: "  {{ .Name }}",
				Selected: "✔ {{ .Name }}",
			},
		}

		index, _, err := prompt.Run()

		if err != nil {
			return "", fmt.Errorf("prompt failed: %w", err)
		}

		return (*examples)[index].Content, nil
	} else if len(*examples) == 1 {
		return (*examples)[0].Content, nil
	}

	return "", fmt.Errorf("no example found for resource %s", resourceType)
}

func getAvailableProviderVersions(registrySource string) ([]string, error) {
	url := fmt.Sprintf("https://registry.terraform.io/v1/providers/%s/versions", registrySource)

	versions, err := util.GetJson[VersionResponse](url)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider versions: %s", err)
	}

	var versionList []string
	for _, v := range versions.Versions {
		versionList = append(versionList, v.Version)
	}

	for i, v := range versionList {
		if v[0] != 'v' {
			versionList[i] = "v" + v
		}
	}

	sort.Slice(versionList, func(i, j int) bool {
		return semver.Compare(versionList[i], versionList[j]) > 0
	})

	return versionList, nil
}
