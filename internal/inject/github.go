package inject

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/google/go-github/v79/github"
	"github.com/phergul/terrasnap/internal/config"
	"github.com/phergul/terrasnap/internal/util"
	"golang.org/x/oauth2"
)

type ExampleClient struct {
	config               *config.Config
	client               *github.Client
	providerMetadata     ProviderMetadata
	specificResourceName string
}
type ProviderMetadata struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Source  string `json:"source"`
	Version string `json:"version"`
}

type ExampleResult struct {
	FileName string
	Path     string
	Content  string
	Name     string
}

type ExampleSearchStrategy string

const (
	StrategyNone          ExampleSearchStrategy = ""
	StrategyResourcesDir  ExampleSearchStrategy = "resources_dir"
	StrategyNamedDir      ExampleSearchStrategy = "named_dir"
	StrategyDirectTFFile  ExampleSearchStrategy = "direct_tf_file"
	StrategyRecursiveScan ExampleSearchStrategy = "recursive_scan"
)

func NewExampleClient(cfg *config.Config) *ExampleClient {
	providerMetadata, err := getProviderRegistryMeta(cfg.Provider.SourceMapping.RegistrySource)
	if err != nil {
		log.Printf("failed to get provider metadata: %v", err)
		return nil
	}

	return &ExampleClient{
		config:           cfg,
		client:           newGithubClient(),
		providerMetadata: providerMetadata,
	}
}

func newGithubClient() *github.Client {
	ctx := context.Background()
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return github.NewClient(nil)
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)
	return client
}

func getProviderRegistryMeta(registrySource string) (ProviderMetadata, error) {
	url := fmt.Sprintf("https://registry.terraform.io/v1/providers/%s", registrySource)

	meta, err := util.GetJson[ProviderMetadata](url)
	if err != nil {
		return ProviderMetadata{}, fmt.Errorf("error getting provider repo: %w", err)
	}

	if meta.Source == "" {
		return ProviderMetadata{}, fmt.Errorf("repository URL not found for provider %s", strings.Split(registrySource, "/")[1])
	}

	return meta, nil
}

func (c *ExampleClient) findGithubExamples(version, resourceType string) (*[]ExampleResult, error) {
	var strategy ExampleSearchStrategy
	if c.config.WorkingStrategy == "" {
		log.Println("(findGithubExamples) no config strategy found; using fallback strategy")
		strategy = StrategyNone
	} else {
		strategy = ExampleSearchStrategy(c.config.WorkingStrategy)
	}

	owner, repo, err := parseGithubURL(c.providerMetadata.Source)
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}

	var opts *github.RepositoryContentGetOptions
	if version != "" {
		opts = &github.RepositoryContentGetOptions{Ref: version}
	}

	_, contents, _, err := c.client.Repositories.GetContents(context.Background(), owner, repo, "examples", opts)
	if err != nil {
		return nil, fmt.Errorf("no examples directory: %w", err)
	}

	if strategy != StrategyNone {
		examples, err := c.tryStrategy(strategy, contents, owner, repo, resourceType, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to find examples for %s: %w", resourceType, err)
		}
		return examples, nil
	}

	strategiesToTry := []ExampleSearchStrategy{
		StrategyResourcesDir,
		StrategyNamedDir,
		StrategyDirectTFFile,
		StrategyRecursiveScan,
	}

	for _, s := range strategiesToTry {
		examples, err := c.tryStrategy(s, contents, owner, repo, resourceType, opts)
		if err != nil {
			log.Printf("%s failed with: %v", s, err)
			continue
		}

		c.config.WorkingStrategy = string(s)
		if err := c.config.WriteConfig(); err != nil {
			log.Printf("github.go: %v", err)
		}

		return examples, nil
	}

	return nil, fmt.Errorf("no example found for resource %s", resourceType)
}

func (c *ExampleClient) tryStrategy(strategy ExampleSearchStrategy, contents []*github.RepositoryContent, owner, repo, resourceType string, opts *github.RepositoryContentGetOptions) (*[]ExampleResult, error) {
	switch strategy {
	case StrategyResourcesDir:
		return c.findInResourcesDir(contents, owner, repo, resourceType, opts)
	case StrategyNamedDir:
		return c.findInNamedDir(contents, owner, repo, resourceType, opts)
	case StrategyDirectTFFile:
		return c.findTFFileInRoot(contents, owner, repo, resourceType, opts)
	case StrategyRecursiveScan:
		return c.recursiveSearch(contents, owner, repo, resourceType, opts)
	}

	return nil, fmt.Errorf("unknown strategy %s", strategy)
}

func (c *ExampleClient) findInResourcesDir(contents []*github.RepositoryContent, owner, repo, resourceType string, opts *github.RepositoryContentGetOptions) (*[]ExampleResult, error) {
	for _, content := range contents {
		if content.GetType() == "dir" && strings.EqualFold(content.GetName(), "resources") {
			_, innerContents, _, err := c.client.Repositories.GetContents(
				context.Background(), owner, repo, content.GetPath(), opts,
			)
			if err != nil {
				return nil, err
			}

			for _, ic := range innerContents {
				if ic.GetType() == "dir" && strings.HasSuffix(strings.ToLower(ic.GetName()), resourceType) {
					log.Println("Found example directory (resources):", ic.GetPath())
					return c.searchExamplesDirectory(owner, repo, ic.GetPath(), resourceType, opts)
				}
			}
		}
	}
	return nil, fmt.Errorf("no example found for resource %s", resourceType)
}

func (c *ExampleClient) findInNamedDir(contents []*github.RepositoryContent, owner, repo, resourceType string, opts *github.RepositoryContentGetOptions) (*[]ExampleResult, error) {
	for _, content := range contents {
		if content.GetType() == "dir" && strings.Contains(strings.ToLower(content.GetName()), resourceType) {
			log.Println("Found example directory:", content.GetPath())
			return c.searchExamplesDirectory(owner, repo, content.GetPath(), resourceType, opts)
		}
	}
	return nil, fmt.Errorf("no example found for resource %s", resourceType)
}

func (c *ExampleClient) findTFFileInRoot(contents []*github.RepositoryContent, owner, repo, resourceType string, opts *github.RepositoryContentGetOptions) (*[]ExampleResult, error) {
	for _, content := range contents {
		if content.GetType() == "file" && strings.HasSuffix(content.GetName(), ".tf") {
			log.Println("Found tf file:", content.GetPath())
			return c.fetchAndValidate(owner, repo, content.GetPath(), resourceType, opts)
		}
	}
	return nil, fmt.Errorf("no example found for resource %s", resourceType)
}

func (c *ExampleClient) recursiveSearch(contents []*github.RepositoryContent, owner, repo, resourceType string, opts *github.RepositoryContentGetOptions) (*[]ExampleResult, error) {
	var totalExamples []ExampleResult
	var errors []error

	for _, citem := range contents {
		if citem.GetType() == "dir" {
			examples, err := c.searchExamplesDirectory(owner, repo, citem.GetPath(), resourceType, opts)
			if err != nil {
				errors = append(errors, fmt.Errorf("[%s] %w", citem.GetPath(), err))
				continue
			}
			if examples != nil {
				totalExamples = append(totalExamples, *examples...)
			}
		}
	}

	if len(totalExamples) == 0 {
		for _, e := range errors {
			log.Printf("recursiveSearch error: %v", e)
		}
		return nil, fmt.Errorf("no examples found for resource %s", resourceType)
	}

	return &totalExamples, nil
}

func parseGithubURL(url string) (owner, repo string, err error) {
	url = strings.TrimSuffix(url, ".git")

	url = strings.TrimPrefix(url, "https://github.com/")
	url = strings.TrimPrefix(url, "http://github.com/")
	url = strings.TrimPrefix(url, "git@github.com:")

	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid Github URL: %s", url)
	}
	return parts[0], parts[1], nil
}

func (c *ExampleClient) searchExamplesDirectory(owner, repo, dirPath, resourceType string, opts *github.RepositoryContentGetOptions) (*[]ExampleResult, error) {
	_, sub, _, err := c.client.Repositories.GetContents(context.Background(), owner, repo, dirPath, opts)
	if err != nil {
		return nil, err
	}

	var totalExamples []ExampleResult
	for _, f := range sub {
		if f.GetType() == "file" && strings.HasSuffix(f.GetName(), ".tf") {
			log.Println("Found tf file:", f.GetName())
			examples, err := c.fetchAndValidate(owner, repo, f.GetPath(), resourceType, opts)
			if err != nil {
				log.Printf("[%s] %s", f.GetName(), err)
				continue
			}
			if examples != nil {
				totalExamples = append(totalExamples, *examples...)
			}
		}
	}

	for _, ex := range totalExamples {
		//dependency found
		if ex.Name == c.specificResourceName {
			return &[]ExampleResult{ex}, nil
		}
	}

	if len(totalExamples) == 0 {
		return nil, fmt.Errorf("no examples found for resource %s", resourceType)
	}
	return &totalExamples, nil
}

func (c *ExampleClient) fetchAndValidate(owner, repo, filePath, resourceType string, opts *github.RepositoryContentGetOptions) (*[]ExampleResult, error) {
	file, _, _, err := c.client.Repositories.GetContents(context.Background(), owner, repo, filePath, opts)
	if err != nil {
		return nil, err
	}

	text, err := file.GetContent()
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(resourceType, c.providerMetadata.Name+"_") {
		resourceType = c.providerMetadata.Name + "_" + resourceType
	}
	re := regexp.MustCompile(fmt.Sprintf(`resource\s+"%s"\s+"[^"]+"\s*{`, regexp.QuoteMeta(resourceType)))
	matches := re.FindAllStringIndex(text, -1)

	if len(matches) == 0 {
		return nil, fmt.Errorf("no resource examples found for %s", resourceType)
	}

	exampleBlocks, err := extractResourceBlocks(text, matches)
	if err != nil {
		return nil, err
	}

	results := make([]ExampleResult, 0, len(matches))
	for _, block := range exampleBlocks {
		name, err := extractResourceName(block, resourceType)
		if err != nil {
			log.Printf("Warning: %v", err)
			name = ""
		}

		results = append(results, ExampleResult{
			FileName: file.GetName(),
			Path:     filePath,
			Content:  block,
			Name:     name,
		})
	}

	log.Printf("Found resource example for %s in %s", resourceType, filePath)
	return &results, nil
}

func extractResourceBlocks(content string, indexes [][]int) ([]string, error) {
	resources := make([]string, 0, len(indexes))
	errors := make([]error, 0, len(indexes))
	for _, index := range indexes {
		start := index[0]

		braceIndex := strings.Index(content[start:], "{")
		if braceIndex == -1 {
			errors = append(errors, fmt.Errorf("malformed resource block, missing '{'"))
			continue
		}

		braceIndex = start + braceIndex

		depth := 0
		end := braceIndex
		for i := braceIndex; i < len(content); i++ {
			c := content[i]

			if c == '{' {
				depth++
			} else if c == '}' {
				depth--
				if depth == 0 {
					end = i + 1
					break
				}
			}
		}

		if depth != 0 {
			errors = append(errors, fmt.Errorf("unterminated resource block"))
			continue
		}

		resources = append(resources, content[start:end])
	}

	if len(resources) == 0 {
		for i, error := range errors {
			log.Printf("Error %d: %v", i, error)
		}
		return nil, fmt.Errorf("no resources extracted (%d errors)", len(errors))
	}

	return resources, nil
}

func extractResourceName(block, resourceType string) (string, error) {
	re := regexp.MustCompile(
		fmt.Sprintf(`resource\s+"%s"\s+"([^"]+)"`, regexp.QuoteMeta(resourceType)),
	)

	match := re.FindStringSubmatch(block)
	if len(match) < 2 {
		return "", fmt.Errorf("could not extract resource name from block")
	}
	return match[1], nil
}
