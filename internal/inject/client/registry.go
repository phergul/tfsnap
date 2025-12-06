package client

import (
	"fmt"
	"log"
	"regexp"
	"slices"
	"strings"

	"github.com/phergul/tfsnap/internal/config"
	"github.com/phergul/tfsnap/internal/util"
)

type RegistryExampleClient struct {
	config               *config.Config
	providerMetadata     util.ProviderMetadata
	Docs                 Docs
	specificResourceName string
}

type RegistryExampleResponse struct {
	Data struct {
		Attributes struct {
			Content string `json:"content"`
		} `json:"attributes"`
	} `json:"data"`
}

type Doc struct {
	Id       string `json:"id"`
	Title    string `json:"title"`
	Category string `json:"category"`
	Path     string `json:"path"`
}

type Docs map[string]Doc

type DocsResponse struct {
	Docs []Doc `json:"docs"`
}

func NewRegistryExampleClient(cfg *config.Config) (ExampleClient, error) {
	providerMetadata, err := util.GetProviderRegistryMeta(cfg.Provider.SourceMapping.RegistrySource)
	if err != nil {
		log.Fatalf("Failed to get provider metadata: %v", err)
		return &RegistryExampleClient{}, err
	}

	return &RegistryExampleClient{
		config:           cfg,
		providerMetadata: providerMetadata,
	}, nil
}

func (c *RegistryExampleClient) SetSpecificResourceName(name string) {
	c.specificResourceName = name
}

func (c *RegistryExampleClient) GetExamples(providerVersion, resourceType string) ([]ExampleResult, error) {
	if c.Docs == nil {
		providerId := fmt.Sprintf("%s/%s", c.providerMetadata.Namespace, c.providerMetadata.Name)
		providerVersion = strings.TrimPrefix(providerVersion, "v")
		docs, err := getRegistryDocs(providerId, providerVersion)
		if err != nil {
			return nil, err
		}

		//remove datasources
		docs = slices.DeleteFunc(docs, func(d Doc) bool {
			return d.Category == "datasource"
		})

		docsMap := make(map[string]Doc, len(docs))
		for _, doc := range docs {
			docsMap[doc.Title] = doc
		}
		c.Docs = docsMap
	}

	if examples, err := c.getResourceExamples(resourceType); err != nil {
		return nil, fmt.Errorf("failed to get examples for resource %s: %w", resourceType, err)
	} else if len(examples) == 0 {
		return nil, fmt.Errorf("no examples found for resource %s", resourceType)
	} else {
		return examples, nil
	}
}

func (c *RegistryExampleClient) GetProviderMetadata() util.ProviderMetadata {
	return c.providerMetadata
}

func init() {
	Register("registry", NewRegistryExampleClient)
}

func getRegistryDocs(providerId, providerVersion string) ([]Doc, error) {
	url := fmt.Sprintf("https://registry.terraform.io/v1/providers/%s/%s", providerId, providerVersion)

	docs, err := util.GetJson[DocsResponse](url)
	if err != nil {
		return nil, fmt.Errorf("failed to get registry docs: %w", err)
	}

	return docs.Docs, nil
}

func (c *RegistryExampleClient) getResourceExamples(resourceType string) ([]ExampleResult, error) {
	if after, ok := strings.CutPrefix(resourceType, c.providerMetadata.Name+"_"); ok {
		resourceType = after
	}

	doc, ok := c.Docs[resourceType]
	if !ok {
		return nil, fmt.Errorf("no documentation found for resource type: %s", resourceType)
	}

	rawExampleContent, err := util.GetJson[RegistryExampleResponse](fmt.Sprintf("https://registry.terraform.io/v2/provider-docs/%s", doc.Id))
	if err != nil {
		return nil, fmt.Errorf("failed to get example content for resource %s: %w", resourceType, err)
	}

	if !strings.HasPrefix(resourceType, c.providerMetadata.Name+"_") {
		resourceType = c.providerMetadata.Name + "_" + resourceType
	}
	if examples, err := extractHCL(rawExampleContent.Data.Attributes.Content, resourceType, c.specificResourceName); err == nil {
		return examples, nil
	} else {
		return nil, fmt.Errorf("failed to extract HCL for resource %s: %w", resourceType, err)
	}
}

func extractHCL(content, resourceType, specificName string) ([]ExampleResult, error) {
	codeBlockRe := regexp.MustCompile("(?s)```terraform\\s*(.*?)\\s*```")
	matches := codeBlockRe.FindAllStringSubmatch(content, -1)

	if len(matches) == 0 {
		return nil, fmt.Errorf("no terraform code blocks found")
	}

	var fullHCL string
	for _, m := range matches {
		fullHCL += m[1] + "\n"
	}

	pattern := fmt.Sprintf(`resource\s+"%s"\s+"([^"]+)"\s*\{`, regexp.QuoteMeta(resourceType))
	resourceRe := regexp.MustCompile(pattern)

	locs := resourceRe.FindAllStringSubmatchIndex(fullHCL, -1)

	if len(locs) == 0 {
		return nil, fmt.Errorf("no examples found for resource %s", resourceType)
	}

	var results []ExampleResult

	for _, loc := range locs {
		blockStart := loc[0]
		braceStart := loc[1]

		nameStart, nameEnd := loc[2], loc[3]
		resourceName := fullHCL[nameStart:nameEnd]

		if specificName != "" && resourceName != specificName {
			continue
		}

		braceCount := 1
		blockEnd := -1

		if braceStart >= len(fullHCL) {
			continue
		}

		for j := braceStart; j < len(fullHCL); j++ {
			char := fullHCL[j]
			if char == '{' {
				braceCount++
			} else if char == '}' {
				braceCount--
			}

			if braceCount == 0 {
				blockEnd = j + 1
				break
			}
		}

		if blockEnd != -1 {
			results = append(results, ExampleResult{
				Name:    resourceName,
				Content: fullHCL[blockStart:blockEnd],
			})

			if specificName != "" {
				return results, nil
			}
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no examples found for resource %s with name %s", resourceType, specificName)
	}

	return results, nil
}
