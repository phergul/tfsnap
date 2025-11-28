package inject

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/phergul/TerraSnap/internal/config"
	"github.com/rogpeppe/go-internal/semver"
)

type TFResource struct {
	Name     string `json:"name"`
	Examples []struct {
		Description string `json:"description"`
		Code        string `json:"code"`
	} `json:"examples"`
}

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

type ProviderVersion struct {
	Version string `json:"version"`
}

type VersionResponse struct {
	Versions []ProviderVersion `json:"versions"`
}

func ValidateResource(schemas *tfjson.ProviderSchemas, registrySource, input string) bool {
	schemaKey := "registry.terraform.io/" + registrySource
	log.Println("schemaKey is:", schemaKey)

	providerSchema, ok := schemas.Schemas[schemaKey]
	if !ok {
		return false
	}

	_, ok = providerSchema.ResourceSchemas[input]
	return ok
}

func InjectResource(cfg *config.Config, resourceType, version string) error {
	tfPath := filepath.Join(cfg.WorkingDirectory, "main.tf")

	resource, err := getResourceExample(cfg.Provider.SourceMapping.RegistrySource, resourceType, version)
	if err != nil {
		log.Println(err)
		return fmt.Errorf("failed to inject resource. Check logs for details.")
	}

	file, err := os.OpenFile(tfPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString("\n" + resource + "\n")
	return err
}

func getResourceExample(registrySource, resourceType, version string) (string, error) {
	versions, err := getAvailableProviderVersions(registrySource)
	if err != nil {
		log.Printf("failed to get provider versions for %s: %v", strings.Split(registrySource, "/")[:1], err)
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

	providerRepoUrl, err := getProviderRepo(registrySource)
	if err != nil {
		return "", fmt.Errorf("failed to get provider repo: %w", err)
	}

	examples, err := getGithubExamples(providerRepoUrl, providerVersion)
	if err != nil {
		return "", fmt.Errorf("failed to get examples from github repo (%s): %w", providerRepoUrl, err)
	}

	// url := fmt.Sprintf("https://registry.terraform.io/providers/%s/%s/docs/resources/%s", registrySource, providerVersion, resourceType)
	//
	// resp, err := http.Get(url)
	// if err != nil {
	// 	return "", err
	// }
	// defer resp.Body.Close()
	//
	// body, _ := io.ReadAll(resp.Body)
	// log.Println(string(body))
	//
	// if resp.StatusCode != http.StatusOK {
	// 	return "", fmt.Errorf("failed to fetch resource: %s", resp.Status)
	// }
	//
	// doc, err := goquery.NewDocumentFromReader(resp.Body)
	// if err != nil {
	// 	return "", err
	// }
	// doc.Find("code.terraform.language-terraform").Each(func(i int, s *goquery.Selection) {
	// 	log.Println("Found code block:", s.Length())
	// })
	//
	// //manually parse the examples from the registry html
	// code := doc.Find("code.terraform.language-terraform").First()
	// if code.Length() == 0 {
	// 	return "", fmt.Errorf("no exmaples found for %s", resourceType)
	// }
	//
	// hcl := code.Text()
	// hcl = strings.TrimSpace(hcl)
	//
	// parts := strings.Split(hcl, `\nresource "`)
	// examples := make([]string, 0, len(parts))
	//
	// for i, part := range parts {
	// 	block := strings.TrimSpace(part)
	// 	if block == "" {
	// 		continue
	// 	}
	//
	// 	if i != 0 {
	// 		block = `resource "` + block
	// 	}
	// 	examples = append(examples, block)
	// }

	if len(examples) == 0 {
		return "", fmt.Errorf("no resource examples found for %s", resourceType)
	}

	if len(examples) > 1 {
		fmt.Printf("Mulitple examples found for %s. Injecting the first.", resourceType)
	}

	// TODO: have some way to select if multiple examples
	return examples[0], nil
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

	var meta ProviderMetadata
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return "", err
	}

	if meta.Source == "" {
		return "", fmt.Errorf("repository URL not found for provider %s", strings.Split(registrySource, "/")[:1])
	}

	return meta.Source, nil
}

func getGithubExamples(repoUrl, version string) ([]string, error) {
	owner, name, err := parseGithubURL(repoUrl)
	if err != nil {
		return nil, err
	}

	githubAPI := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/examples", owner, name)
	if version != "" {
		githubAPI += "?ref=" + version
	}

	log.Println("github api url:", githubAPI)
	resp, err := http.Get(githubAPI)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch github api: %s", resp.Status)
	}

	var contents []GitHubContent
	if err := json.NewDecoder(resp.Body).Decode(&contents); err != nil {
		return nil, err
	}

	log.Println(contents)

	return nil, nil
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

func getAvailableProviderVersions(registrySource string) ([]string, error) {
	url := fmt.Sprintf("https://registry.terraform.io/v1/providers/%s/versions", registrySource)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get provider versions: %s", resp.Status)
	}

	var versions VersionResponse
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return nil, err
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

	// for _, v := range versionList {
	// 	log.Println(v)
	// }

	return versionList, nil
}
