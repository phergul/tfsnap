package client

import (
	"testing"

	"github.com/phergul/tfsnap/internal/config"
	"github.com/phergul/tfsnap/internal/util"
)

func TestRegister(t *testing.T) {
	testName := "test-client"
	testConstructor := func(cfg *config.Config) (ExampleClient, error) {
		return &mockClient{}, nil
	}

	Register(testName, testConstructor)

	constructor, ok := clientRegistry[testName]
	if !ok {
		t.Errorf("Failed to register client %q", testName)
	}

	if constructor == nil {
		t.Error("Registered constructor is nil")
	}
}

func TestNew(t *testing.T) {
	testName := "test-client-new"
	mockCfg := &config.Config{}
	testConstructor := func(cfg *config.Config) (ExampleClient, error) {
		return &mockClient{}, nil
	}

	Register(testName, testConstructor)

	client, err := New(testName, mockCfg)
	if err != nil {
		t.Errorf("New failed with registered client: %v", err)
	}

	if client == nil {
		t.Error("New returned nil client")
	}
}

func TestNewUnknownClient(t *testing.T) {
	_, err := New("unknown-client-that-does-not-exist", &config.Config{})
	if err == nil {
		t.Error("New should return error for unknown client type")
	}
}

type mockClient struct{}

func (m *mockClient) GetExamples(providerVersion, resourceType string) ([]ExampleResult, error) {
	return []ExampleResult{}, nil
}

func (m *mockClient) SetSpecificResourceName(name string) {}

func (m *mockClient) GetProviderMetadata() util.ProviderMetadata {
	return util.ProviderMetadata{}
}
