package inject

import (
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strings"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/phergul/terrasnap/internal/config"
)

func InjectSkeleton(cfg *config.Config, schema *tfjson.Schema, resourceType string) error {
	tfPath := filepath.Join(cfg.WorkingDirectory, "main.tf")

	resource, err := buildSkeleton(schema, resourceType)
	if err != nil {
		return fmt.Errorf("error building skeleton for %s", resourceType)
	}

	return writeResourceToFile(tfPath, resource)
}

func buildSkeleton(schema *tfjson.Schema, resourceType string) (string, error) {
	var resource strings.Builder

	resource.WriteString(fmt.Sprintf("resource \"%s\" \"test\" {\n", resourceType))
	resource.WriteString(renderBlock(schema.Block, 1))
	resource.WriteString("}\n")

	return resource.String(), nil
}

func renderBlock(block *tfjson.SchemaBlock, indent int) string {
	var out strings.Builder
	indentStr := strings.Repeat("\t", indent)

	attrKeys := make([]string, 0, len(block.Attributes))
	for k := range block.Attributes {
		attrKeys = append(attrKeys, k)
	}
	sort.Strings(attrKeys)

	for _, key := range attrKeys {
		attr := block.Attributes[key]
		if attr.Deprecated || key == "id" {
			continue
		}
		if attr.Required || attr.Optional {
			out.WriteString(getEmptyValue(key, *attr, indent))
		}
	}

	nestedKeys := make([]string, 0, len(block.NestedBlocks))
	for k := range block.NestedBlocks {
		nestedKeys = append(nestedKeys, k)
	}
	sort.Strings(nestedKeys)

	for _, name := range nestedKeys {
		nested := block.NestedBlocks[name]
		out.WriteString(fmt.Sprintf("%s%s {\n", indentStr, name))
		out.WriteString(renderBlock(nested.Block, indent+1))
		out.WriteString(fmt.Sprintf("%s}\n", indentStr))
	}

	return out.String()
}

func getEmptyValue(key string, v tfjson.SchemaAttribute, indent int) string {
	log.Printf("%s::%s [R: %t] [O: %t] [C: %t]", key, v.AttributeType.FriendlyName(), v.Required, v.Optional, v.Computed)
	indentStr := strings.Repeat("\t", indent)

	switch v.AttributeType.FriendlyName() {
	case "string":
		log.Printf("%s, with indent %d", key, indent)
		return fmt.Sprintf("%s%s = \"\"\n", indentStr, key)
	case "bool":
		log.Printf("%s, with indent %d", key, indent)
		return fmt.Sprintf("%s%s = false\n", indentStr, key)
	case "number":
		log.Printf("%s, with indent %d", key, indent)
		return fmt.Sprintf("%s%s = 0\n", indentStr, key)
	case "list of string", "set of string":
		log.Printf("%s, with indent %d", key, indent)
		return fmt.Sprintf("%s%s = [ ]\n", indentStr, key)
	case "object", "list of object", "set of object":
		log.Printf("%s, with indent %d", key, indent)
		var out strings.Builder
		out.WriteString(fmt.Sprintf("%s%s {\n", indentStr, key))

		elemType := v.AttributeType.ElementType()
		nestedKeys := make([]string, 0, len(elemType.AttributeTypes()))
		for k := range elemType.AttributeTypes() {
			nestedKeys = append(nestedKeys, k)
		}
		sort.Strings(nestedKeys)

		for _, nestedKey := range nestedKeys {
			nestedType := elemType.AttributeTypes()[nestedKey]
			nestedAttr := &tfjson.SchemaAttribute{
				AttributeType: nestedType,
				Required:      v.Required,
				Optional:      v.Optional,
			}
			out.WriteString(getEmptyValue(nestedKey, *nestedAttr, indent+1))
		}

		out.WriteString(fmt.Sprintf("%s}\n", indentStr))
		return out.String()
	default:
		log.Printf("UNKNOWN TYPE?: %s", v.AttributeType.FriendlyName())
		return ""
	}
}

