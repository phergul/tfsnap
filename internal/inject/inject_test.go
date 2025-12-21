package inject_test

import (
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/phergul/tfsnap/internal/inject"
	"github.com/phergul/tfsnap/internal/inject/client"
	"github.com/phergul/tfsnap/internal/util"
	"github.com/zclconf/go-cty/cty"
)

var testSchema = &tfjson.ProviderSchema{
	ResourceSchemas: map[string]*tfjson.Schema{
		"name_to_validate": {
			Block: &tfjson.SchemaBlock{
				Attributes: map[string]*tfjson.SchemaAttribute{
					"id": {
						AttributeType: cty.String,
						Required:      true,
					},
				},
			},
		},
	},
}

func TestValidateResource(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		schema *tfjson.ProviderSchema
		input  string
		want   *tfjson.Schema
		want2  bool
	}{
		{
			name:   "Valid resource name",
			schema: testSchema,
			input:  "name_to_validate",
			want:   testSchema.ResourceSchemas["name_to_validate"],
			want2:  true,
		},
		{
			name:   "Invalid resource name",
			schema: testSchema,
			input:  "invalid_name",
			want:   nil,
			want2:  false,
		},
		{
			name:   "Empty schema",
			schema: &tfjson.ProviderSchema{ResourceSchemas: make(map[string]*tfjson.Schema)},
			input:  "any_resource",
			want:   nil,
			want2:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got2 := inject.ValidateResource(tt.schema, tt.input)
			if got != tt.want {
				t.Errorf("ValidateResource() = %v, want %v", got, tt.want)
			}
			if got2 != tt.want2 {
				t.Errorf("ValidateResource() = %v, want %v", got2, tt.want2)
			}
		})
	}
}

func TestNewDependencyResolver(t *testing.T) {
	mockClient := &mockExampleClient{}
	resolver := inject.NewDependencyResolver(mockClient)

	if resolver == nil {
		t.Error("NewDependencyResolver returned nil")
	}
}

// Mock implementation of ExampleClient interface for testing
type mockExampleClient struct{}

func (m *mockExampleClient) GetExamples(providerVersion, resourceType string) ([]client.ExampleResult, error) {
	return []client.ExampleResult{}, nil
}

func (m *mockExampleClient) SetSpecificResourceName(name string) {}

func (m *mockExampleClient) GetProviderMetadata() util.ProviderMetadata {
	return util.ProviderMetadata{}
}
