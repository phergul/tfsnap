package util

import (
	"fmt"
	"log"
	"sort"

	"github.com/phergul/tfsnap/internal/config"
	"golang.org/x/mod/semver"
)

type ProviderVersion struct {
	Version string `json:"version"`
}

type VersionResponse struct {
	Versions []ProviderVersion `json:"versions"`
}

func GetAvailableProviderVersions(registrySource string) ([]string, error) {
	url := fmt.Sprintf("https://registry.terraform.io/v1/providers/%s/versions", registrySource)

	versions, err := GetJson[VersionResponse](url)
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

func GetLatestProviderVersion(cfg *config.Config) string {
	versions, err := GetAvailableProviderVersions(cfg.Provider.SourceMapping.RegistrySource)
	if err != nil {
		log.Printf("failed to get available provider versions: %v", err)
		return ""
	}
	if len(versions) == 0 {
		log.Println("no available provider versions found")
		return ""
	}
	return versions[0]
}
