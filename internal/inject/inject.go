package inject

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/manifoldco/promptui"
	"github.com/phergul/tfsnap/internal/config"
	"github.com/phergul/tfsnap/internal/inject/client"
	"github.com/phergul/tfsnap/internal/util"
)

func ValidateResource(schema *tfjson.ProviderSchema, input string) (*tfjson.Schema, bool) {
	resourceSchema, ok := schema.ResourceSchemas[input]
	if !ok {
		return nil, ok
	}
	return resourceSchema, ok
}

func InjectResource(cfg *config.Config, resourceType, version string, dependency bool) error {
	tfPath := filepath.Join(cfg.WorkingDirectory, "main.tf")

	resources, err := getResourceExampleWithDependencies(cfg, resourceType, version, dependency)
	if err != nil {
		log.Println(err)
		return fmt.Errorf("failed to inject resource. Check logs for details.")
	}

	for _, resource := range resources {
		existingContent, err := os.ReadFile(tfPath)
		if err != nil && !os.IsNotExist(err) {
			log.Println(err)
			return fmt.Errorf("failed to read existing file. Check logs for details.")
		}

		if resourceAlreadyExists(string(existingContent), resource) {
			log.Printf("Resource already exists in file, skipping duplicate injection")
			continue
		}
		if err := writeResourceToFile(tfPath, resource); err != nil {
			log.Println(err)
			return fmt.Errorf("failed to inject resource. Check logs for details.")
		}
	}

	return nil
}

func resourceAlreadyExists(existingContent, newResource string) bool {
	return strings.Contains(existingContent, strings.TrimSpace(newResource))
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

func getResourceExampleWithDependencies(cfg *config.Config, resourceType, version string, dependency bool) ([]string, error) {
	versions, err := util.GetAvailableProviderVersions(cfg.Provider.SourceMapping.RegistrySource)
	if err != nil {
		log.Printf("failed to get provider versions for %s: %v", strings.Split(cfg.Provider.SourceMapping.RegistrySource, "/")[:1], err)
		return nil, err
	}

	var providerVersion string
	if version == "" {
		providerVersion = versions[0] //latest
	} else {
		if !slices.Contains(versions, version) {
			return nil, fmt.Errorf("provided version %s does not exist for provider", version)
		}
		providerVersion = version
	}

	clientType := cfg.ExampleClientType
	if clientType == "" {
		clientType = "github"
	}

	examplesClient, err := client.New(clientType, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create examples client: %w", err)
	}

	examples, err := examplesClient.GetExamples(providerVersion, resourceType)
	if err != nil {
		return nil, fmt.Errorf("failed to get examples: %w", err)
	}

	var initialResource string
	if len(examples) > 1 {
		fmt.Printf("Multiple %s resources found\n", resourceType)
		prompt := promptui.Select{
			Label: fmt.Sprintf("Select %s example to inject", resourceType),
			Items: examples,
			Templates: &promptui.SelectTemplates{
				Label:    "{{ . }}:",
				Active:   "> {{ .Name | underline }}",
				Inactive: "  {{ .Name }}",
				Selected: "✔ {{ .Name }}",
			},
		}

		index, _, err := prompt.Run()

		if err != nil {
			return nil, fmt.Errorf("prompt failed: %w", err)
		}

		initialResource = examples[index].Content
	} else if len(examples) == 1 {
		initialResource = examples[0].Content
	} else {
		return nil, fmt.Errorf("no example found for resource %s", resourceType)
	}

	if dependency {
		resolver := NewDependencyResolver(examplesClient)

		log.Println("Checking dependencies for resource:", resourceType)
		visited := make(map[string]bool)
		resolvedDeps := []resolvedDependency{}

		resolver.resolveDependenciesRecursive(initialResource, visited, &resolvedDeps)

		resources := make([]string, 0, len(resolvedDeps)+1)
		for _, dep := range resolvedDeps {
			resources = append(resources, dep.content)
		}
		resources = append(resources, initialResource)

		return resources, nil
	}

	return []string{initialResource}, nil
}
