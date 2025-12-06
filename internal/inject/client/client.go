package client

import (
	"fmt"

	"github.com/phergul/tfsnap/internal/config"
	"github.com/phergul/tfsnap/internal/util"
)

type ExampleResult struct {
	Content string
	Name    string
}

type ExampleClient interface {
	GetExamples(providerVersion, resourceType string) ([]ExampleResult, error)
	SetSpecificResourceName(name string)
	GetProviderMetadata() util.ProviderMetadata
}

type clientConstructor func(cfg *config.Config) (ExampleClient, error)

var clientRegistry = make(map[string]clientConstructor)

func Register(name string, constructor clientConstructor) {
	clientRegistry[name] = constructor
}

func New(name string, cfg *config.Config) (ExampleClient, error) {
	constructor, ok := clientRegistry[name]
	if !ok {
		return nil, fmt.Errorf("unknown example client type: %s", name)
	}
	return constructor(cfg)
}
