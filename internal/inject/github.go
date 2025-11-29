package inject

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/google/go-github/v79/github"
	"golang.org/x/oauth2"
)

type ProviderMetadata struct {
	ID        string `json:"id"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Source    string `json:"source"`
}

type GitHubContent struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	DownloadURL string `json:"download_url"`
	Type        string `json:"type"`
}

type ExampleResult struct {
	FileName string
	Path     string
	Content  string
}

var meta ProviderMetadata
var opts *github.RepositoryContentGetOptions

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

func getProviderRepo(registrySource string) (string, error) {
	url := fmt.Sprintf("https://registry.terraform.io/v1/providers/%s", registrySource)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to fetch provider metadata: %s", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return "", err
	}

	if meta.Source == "" {
		return "", fmt.Errorf("repository URL not found for provider %s", strings.Split(registrySource, "/")[:1])
	}

	return meta.Source, nil
}

func findGithubExample(repoUrl, version, resourceType string) (*ExampleResult, error) {
	owner, repo, err := parseGithubURL(repoUrl)
	if err != nil {
		return nil, err
	}

	client := newGithubClient()

	if version != "" {
		opts = &github.RepositoryContentGetOptions{Ref: version}
	}

	_, contents, _, err := client.Repositories.GetContents(
		context.Background(), owner, repo, "examples", opts,
	)
	if err != nil {
		return nil, fmt.Errorf("no examples directory: %w", err)
	}

	for _, c := range contents {
		if c.GetType() == "dir" && strings.EqualFold(c.GetName(), "resources") {
			_, innerContents, _, err := client.Repositories.GetContents(
				context.Background(), owner, repo, c.GetPath(), opts,
			)
			if err != nil {
				return nil, err
			}

			for _, ic := range innerContents {
				if ic.GetType() == "dir" && strings.HasSuffix(strings.ToLower(ic.GetName()), resourceType) {
					log.Println("Found example directory (resources):", ic.GetPath())
					return searchExamplesDirectory(client, owner, repo, ic.GetPath(), resourceType)
				}
			}
		}
	}

	for _, c := range contents {
		if c.GetType() == "dir" && strings.Contains(strings.ToLower(c.GetName()), resourceType) {

			log.Println("Found example directory:", c.GetPath())
			return searchExamplesDirectory(client, owner, repo, c.GetPath(), resourceType)
		}
	}

	for _, c := range contents {
		if c.GetType() == "file" && strings.HasSuffix(c.GetName(), ".tf") && strings.Contains(strings.ToLower(c.GetName()), resourceType) {
			log.Println("Found example file:", c.GetPath())
			return fetchAndValidate(client, owner, repo, c.GetPath(), resourceType)
		}
	}

	for _, c := range contents {
		if c.GetType() == "dir" {
			ex, _ := searchExamplesDirectory(client, owner, repo, c.GetPath(), resourceType)
			if ex != nil {
				return ex, nil
			}
		}
	}

	return nil, fmt.Errorf("no example found for resource %s", resourceType)
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

func searchExamplesDirectory(client *github.Client, owner, repo, dirPath, resourceType string) (*ExampleResult, error) {
	_, sub, _, err := client.Repositories.GetContents(context.Background(), owner, repo, dirPath, opts)
	if err != nil {
		return nil, err
	}

	for _, f := range sub {
		if f.GetType() == "file" && strings.HasSuffix(f.GetName(), ".tf") {
			log.Println("Found tf file:", f.GetName())
			ex, err := fetchAndValidate(client, owner, repo, f.GetPath(), resourceType)
			if err != nil {
				log.Printf("[%s] %s", f.GetName(), err)
				continue
			}
			if ex != nil {
				return ex, nil
			}
		}
	}
	return nil, fmt.Errorf("no resource example inside %s", dirPath)
}

func fetchAndValidate(client *github.Client, owner, repo, filePath, resourceType string) (*ExampleResult, error) {
	file, _, _, err := client.Repositories.GetContents(context.Background(), owner, repo, filePath, opts)
	if err != nil {
		return nil, err
	}

	text, err := file.GetContent()
	if err != nil {
		return nil, err
	}

	resourceType = meta.Name + "_" + resourceType
	re := regexp.MustCompile(fmt.Sprintf(`resource\s+"%s"\s+"[^"]+"\s*{`, regexp.QuoteMeta(resourceType)))
	matches := re.FindAllStringIndex(text, -1)

	if len(matches) == 0 {
		return nil, fmt.Errorf("no resource examples found for %s", resourceType)
	}

	if len(matches) > 1 {
		log.Printf(
			`Found %d resources for "%s" examples in %s; using the first one.`,
			len(matches), resourceType, file.GetName(),
		)
	}

	// TODO: have some way to choose which example when there are mulitple
	block, err := extractFirstResourceBlock(text, resourceType)
	if err != nil {
		return nil, err
	}

	log.Printf("Found resource example for %s in %s", resourceType, filePath)
	return &ExampleResult{
		FileName: file.GetName(),
		Path:     filePath,
		Content:  block,
	}, nil
}

func extractFirstResourceBlock(content, resourceType string) (string, error) {
	target := fmt.Sprintf(`resource "%s"`, resourceType)
	idx := strings.Index(content, target)
	if idx == -1 {
		return "", fmt.Errorf("resource %s not found", resourceType)
	}

	start := idx

	braceIndex := strings.Index(content[start:], "{")
	if braceIndex == -1 {
		return "", fmt.Errorf("malformed resource block, missing '{'")
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
		return "", fmt.Errorf("unterminated resource block")
	}

	return content[start:end], nil
}
