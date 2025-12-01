package inject

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func (c *ExampleClient) checkDependencies(resource string) ([]string, error) {
	dependencies, err := extractDependencies(resource, c.providerMetadata.Name)
	if err != nil {
		return nil, fmt.Errorf("error extracting resource attribute keys: %w", err)
	}

	return dependencies, nil
}

func (c *ExampleClient) resolveDependencies(collectedDependencies *[]string, dependencies []string) {
	var resolved []string
	for _, dep := range dependencies {
		parts := strings.SplitN(dep, ".", 2)
		c.specificResourceName = parts[1]
		example, err := c.findGithubExamples(c.providerMetadata.Version, parts[0])
		if err != nil {
			fmt.Printf("Dependency %s could not be resolved. Skipping...\n", dep)
			log.Printf("error finding GitHub example for %s: %v", dep, err)
			continue
		}
		fmt.Printf("Resolved dependency %s\n", dep)
		resolved = append(resolved, (*example)[0].Content)
	}
	*collectedDependencies = append(*collectedDependencies, resolved...)
}

func extractDependencies(resource, providerPrefix string) ([]string, error) {
	f, diags := hclsyntax.ParseConfig([]byte(resource), "dependency.tf", hcl.Pos{})
	if diags.HasErrors() {
		return nil, fmt.Errorf("parse error: %v", diags)
	}

	var deps []string
	walkBody(f.Body.(*hclsyntax.Body), providerPrefix, &deps)

	log.Printf("dep from ast: %v", deps)
	return deps, nil
}

func walkBody(body *hclsyntax.Body, providerPrefix string, deps *[]string) {
	for _, attr := range body.Attributes {
		log.Printf("walking expr")
		walkExpr(attr.Expr, providerPrefix, deps)
	}

	for _, block := range body.Blocks {
		log.Printf("walking block")
		walkBody(block.Body, providerPrefix, deps)
	}
}

func walkExpr(expr hclsyntax.Expression, providerPrefix string, deps *[]string) {
	switch e := expr.(type) {
	case *hclsyntax.ScopeTraversalExpr:
		log.Printf("scope traversal")
		ref := traversalToString(e.Traversal)
		if strings.HasPrefix(ref, providerPrefix) {
			*deps = append(*deps, ref)
		}
	case *hclsyntax.RelativeTraversalExpr:
		log.Printf("relative traversal")
		ref := traversalToString(e.Traversal)
		if strings.HasPrefix(ref, providerPrefix) {
			*deps = append(*deps, ref)
		}
	case *hclsyntax.TupleConsExpr:
		for _, elem := range e.Exprs {
			log.Printf("walking tuple element")
			walkExpr(elem, providerPrefix, deps)
		}
	case *hclsyntax.ObjectConsExpr:
		for _, item := range e.Items {
			log.Printf("walking object key")
			walkExpr(item.KeyExpr, providerPrefix, deps)
			log.Printf("walking object value")
			walkExpr(item.ValueExpr, providerPrefix, deps)
		}
	case *hclsyntax.FunctionCallExpr:
		for _, arg := range e.Args {
			log.Printf("walking function argument")
			walkExpr(arg, providerPrefix, deps)
		}
	case *hclsyntax.TemplateExpr:
		for _, part := range e.Parts {
			log.Printf("walking template part")
			walkExpr(part, providerPrefix, deps)
		}
	case *hclsyntax.TemplateWrapExpr:
		log.Printf("walking template wrap")
		walkExpr(e.Wrapped, providerPrefix, deps)
	case *hclsyntax.ParenthesesExpr:
		log.Printf("walking parentheses")
		walkExpr(e.Expression, providerPrefix, deps)
	}
}

func traversalToString(t hcl.Traversal) string {
	parts := []string{}
	for _, step := range t {
		switch s := step.(type) {
		case hcl.TraverseRoot:
			log.Printf("traversing root: %s", s.Name)
			// TODO: include data source resolution
			if s.Name == "data" {
				continue
			}
			parts = append(parts, s.Name)
		case hcl.TraverseAttr:
			log.Printf("traversing attribute: %s", s.Name)
			if s.Name == "id" {
				continue
			}
			parts = append(parts, s.Name)
		}
	}
	return strings.Join(parts, ".")
}
