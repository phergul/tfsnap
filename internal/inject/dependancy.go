package inject

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

type resolvedDependency struct {
	name    string
	content string
}

func (c *ExampleClient) checkDependencies(resource string) ([]string, error) {
	dependencies, err := extractDependencies(resource, c.providerMetadata.Name)
	if err != nil {
		return nil, fmt.Errorf("error extracting resource attribute keys: %w", err)
	}

	return dependencies, nil
}

func (c *ExampleClient) resolveDependenciesRecursive(resource string, visited map[string]bool, resolvedResources *[]resolvedDependency) {
	deps, err := c.checkDependencies(resource)
	if err != nil {
		fmt.Printf("failed to check dependencies; skipping...\n")
		log.Printf("error checking dependencies: %v", err)
		return
	}

	if len(deps) == 0 {
		log.Println("No dependencies found for this resource")
		return
	}

	fmt.Printf("Found %d dependencies. Resolving...\n", len(deps))
	log.Printf("Found dependencies: %v", deps)

	trueResolveCount := len(deps)
	for _, depName := range deps {
		if visited[depName] {
			log.Printf("Dependency %s already resolved, skipping...\n", depName)
			trueResolveCount--
			continue
		}
		visited[depName] = true

		parts := strings.SplitN(depName, ".", 2)
		c.specificResourceName = parts[1]
		example, err := c.findGithubExamples(c.providerMetadata.Version, parts[0])
		if err != nil {
			fmt.Printf("Dependency %s could not be resolved. Skipping...\n", depName)
			log.Printf("error finding GitHub example for %s: %v", depName, err)
			continue
		}

		resourceContent := (*example)[0].Content
		fmt.Printf("Resolved dependency %s\n", depName)

		c.resolveDependenciesRecursive(resourceContent, visited, resolvedResources)

		*resolvedResources = append(*resolvedResources, resolvedDependency{
			name:    depName,
			content: resourceContent,
		})
	}
	fmt.Printf("Resolved %d dependencies\n", trueResolveCount)
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
		walkExpr(attr.Expr, providerPrefix, deps)
	}

	for _, block := range body.Blocks {
		walkBody(block.Body, providerPrefix, deps)
	}
}

func walkExpr(expr hclsyntax.Expression, providerPrefix string, deps *[]string) {
	switch e := expr.(type) {
	case *hclsyntax.ScopeTraversalExpr:
		ref := traversalToString(e.Traversal)
		if strings.HasPrefix(ref, providerPrefix) {
			*deps = append(*deps, ref)
		}
	case *hclsyntax.RelativeTraversalExpr:
		ref := traversalToString(e.Traversal)
		if strings.HasPrefix(ref, providerPrefix) {
			*deps = append(*deps, ref)
		}
	case *hclsyntax.TupleConsExpr:
		for _, elem := range e.Exprs {
			walkExpr(elem, providerPrefix, deps)
		}
	case *hclsyntax.ObjectConsExpr:
		for _, item := range e.Items {
			walkExpr(item.KeyExpr, providerPrefix, deps)
			walkExpr(item.ValueExpr, providerPrefix, deps)
		}
	case *hclsyntax.FunctionCallExpr:
		for _, arg := range e.Args {
			walkExpr(arg, providerPrefix, deps)
		}
	case *hclsyntax.TemplateExpr:
		for _, part := range e.Parts {
			walkExpr(part, providerPrefix, deps)
		}
	case *hclsyntax.TemplateWrapExpr:
		walkExpr(e.Wrapped, providerPrefix, deps)
	case *hclsyntax.ParenthesesExpr:
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
