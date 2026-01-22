package template

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/phergul/tfsnap/internal/config"
	"github.com/phergul/tfsnap/internal/tui"
)

func templatesRoot(cfg *config.Config) string {
	return filepath.Join(cfg.WorkingDirectory, ".tfsnap", "templates", cfg.Provider.Name)
}

func SaveTemplate(cfg *config.Config, resourceType, resourceName, templateName string) (string, error) {
	sourceFile := filepath.Join(cfg.WorkingDirectory, "main.tf")

	data, err := os.ReadFile(sourceFile)
	if err != nil {
		return "", fmt.Errorf("read source file: %w", err)
	}

	file, diag := hclsyntax.ParseConfig(data, sourceFile, hcl.InitialPos)
	if diag.HasErrors() {
		return "", fmt.Errorf("parse hcl: %s", diag.Error())
	}

	body := file.Body.(*hclsyntax.Body)
	var blockRange hcl.Range
	found := false
	for _, b := range body.Blocks {
		if b.Type == "resource" && len(b.Labels) == 2 && b.Labels[0] == resourceType && b.Labels[1] == resourceName {
			blockRange = b.Range()
			found = true
			break
		}
	}
	if !found {
		return "", fmt.Errorf("resource %q.%q not found in %s", resourceType, resourceName, sourceFile)
	}

	start, end := blockRange.Start.Byte, blockRange.End.Byte
	if start < 0 || end > len(data) || start >= end {
		return "", fmt.Errorf("invalid block range")
	}
	content := string(data[start:end])

	if templateName == "" {
		templateName = resourceName
	}
	dir := filepath.Join(templatesRoot(cfg), resourceType)
	if err := ensureDir(dir); err != nil {
		return "", fmt.Errorf("ensure template dir: %w", err)
	}
	outPath := filepath.Join(dir, templateName+".tf")
	if err := os.WriteFile(outPath, []byte(strings.TrimSpace(content)+"\n"), 0o644); err != nil {
		return "", fmt.Errorf("write template: %w", err)
	}
	return outPath, nil
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

type ResourceInfo struct {
	Type string
	Name string
}

func RunSave(cfg *config.Config, templateName string) error {
	resources, err := listResourcesInMainTf(cfg)
	if err != nil {
		return fmt.Errorf("failed to list resources: %w", err)
	}

	if len(resources) == 0 {
		fmt.Println("No resources found in main.tf")
		return nil
	}

	items := make([]tui.Item, len(resources))
	for i, r := range resources {
		content, err := getResourceContent(cfg, r.Type, r.Name)
		if err != nil {
			return fmt.Errorf("failed to get resource content: %w", err)
		}
		items[i] = tui.Item{
			Label:   fmt.Sprintf("%s.%s", r.Type, r.Name),
			Content: content,
			Meta:    r,
		}
	}

	selected, err := tui.RunSelector("Select Resource to Save as Template", items)
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	if selected == nil {
		fmt.Println("No resource selected.")
		return nil
	}

	selectedResource, ok := selected.Meta.(ResourceInfo)
	if !ok {
		return fmt.Errorf("invalid resource selected")
	}
	outPath, err := SaveTemplate(cfg, selectedResource.Type, selectedResource.Name, templateName)
	if err != nil {
		return fmt.Errorf("failed to save template: %w", err)
	}

	fmt.Printf("✔ Template saved to: %s\n", outPath)
	return nil
}

func Run(cfg *config.Config) error {
	templatesDir := templatesRoot(cfg)

	if !dirExists(templatesDir) {
		fmt.Println("No templates found. Create templates first with 'tfsnap template save <name>'.")
		return nil
	}

	templates, err := loadAllTemplates(templatesDir)
	if err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}

	if len(templates) == 0 {
		fmt.Println("No templates found.")
		return nil
	}

	items := make([]tui.Item, len(templates))
	for i, tmpl := range templates {
		items[i] = tui.Item{
			Label:   fmt.Sprintf("%s / %s", tmpl.ResourceType, tmpl.Name),
			Content: tmpl.Content,
			Meta:    tmpl,
		}
	}

	actions := []tui.Action{
		{Key: "enter", Label: "inject", Description: "Inject template into main.tf"},
		{Key: "d", Label: "delete", Description: "Delete this template"},
	}

	result, err := tui.RunActionSelector("Templates", items, actions)
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	if result == nil {
		return nil
	}

	tmpl, ok := result.Item.Meta.(TemplateItem)
	if !ok {
		return fmt.Errorf("invalid template selected")
	}

	switch result.Action {
	case "enter":
		if err := injectTemplate(cfg, tmpl); err != nil {
			return fmt.Errorf("failed to inject template: %w", err)
		}
		fmt.Printf("✔ Template '%s' injected successfully!\n", tmpl.Name)

	case "d":
		if err := os.Remove(tmpl.Path); err != nil {
			return fmt.Errorf("failed to remove template: %w", err)
		}
		fmt.Printf("✔ Template '%s' deleted successfully!\n", tmpl.Name)
	}

	return nil
}

func RunList(cfg *config.Config) error {
	return Run(cfg)
}

func RunRemove(cfg *config.Config, templateName string) error {
	templatesDir := templatesRoot(cfg)

	if !dirExists(templatesDir) {
		fmt.Println("No templates found.")
		return nil
	}

	templatePath, err := findTemplate(templatesDir, templateName)
	if err != nil {
		return fmt.Errorf("failed to find template: %w", err)
	}

	if templatePath == "" {
		fmt.Printf("Template '%s' not found.\n", templateName)
		return nil
	}

	if err := os.Remove(templatePath); err != nil {
		return fmt.Errorf("failed to remove template: %w", err)
	}

	fmt.Printf("✔ Template '%s' removed successfully.\n", templateName)
	return nil
}

func listResourcesInMainTf(cfg *config.Config) ([]ResourceInfo, error) {
	sourceFile := filepath.Join(cfg.WorkingDirectory, "main.tf")

	data, err := os.ReadFile(sourceFile)
	if err != nil {
		return nil, fmt.Errorf("read source file: %w", err)
	}

	file, diag := hclsyntax.ParseConfig(data, sourceFile, hcl.InitialPos)
	if diag.HasErrors() {
		return nil, fmt.Errorf("parse hcl: %s", diag.Error())
	}

	body := file.Body.(*hclsyntax.Body)
	var resources []ResourceInfo

	for _, b := range body.Blocks {
		if b.Type == "resource" && len(b.Labels) == 2 {
			resources = append(resources, ResourceInfo{
				Type: b.Labels[0],
				Name: b.Labels[1],
			})
		}
	}

	return resources, nil
}

func getResourceContent(cfg *config.Config, resourceType, resourceName string) (string, error) {
	sourceFile := filepath.Join(cfg.WorkingDirectory, "main.tf")

	data, err := os.ReadFile(sourceFile)
	if err != nil {
		return "", fmt.Errorf("read source file: %w", err)
	}

	file, diag := hclsyntax.ParseConfig(data, sourceFile, hcl.InitialPos)
	if diag.HasErrors() {
		return "", fmt.Errorf("parse hcl: %s", diag.Error())
	}

	body := file.Body.(*hclsyntax.Body)
	for _, b := range body.Blocks {
		if b.Type == "resource" && len(b.Labels) == 2 && b.Labels[0] == resourceType && b.Labels[1] == resourceName {
			start, end := b.Range().Start.Byte, b.Range().End.Byte
			if start >= 0 && end <= len(data) && start < end {
				return string(data[start:end]), nil
			}
		}
	}

	return "", fmt.Errorf("resource not found")
}

func findTemplate(templatesDir, templateName string) (string, error) {
	var foundPath string

	err := filepath.Walk(templatesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, "/"+templateName+".tf") {
			foundPath = path
			return filepath.SkipAll
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return foundPath, nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

type TemplateItem struct {
	ResourceType string
	Name         string
	Path         string
	Content      string
}

func loadAllTemplates(templatesDir string) ([]TemplateItem, error) {
	var templates []TemplateItem

	resourceTypes, err := os.ReadDir(templatesDir)
	if err != nil {
		return nil, err
	}

	for _, rt := range resourceTypes {
		if !rt.IsDir() {
			continue
		}

		resourceType := rt.Name()
		typeDir := filepath.Join(templatesDir, resourceType)

		files, err := os.ReadDir(typeDir)
		if err != nil {
			continue
		}

		for _, f := range files {
			if !f.IsDir() && strings.HasSuffix(f.Name(), ".tf") {
				templatePath := filepath.Join(typeDir, f.Name())
				content, err := os.ReadFile(templatePath)
				if err != nil {
					continue
				}

				templates = append(templates, TemplateItem{
					ResourceType: resourceType,
					Name:         strings.TrimSuffix(f.Name(), ".tf"),
					Path:         templatePath,
					Content:      string(content),
				})
			}
		}
	}

	return templates, nil
}

func injectTemplate(cfg *config.Config, tmpl TemplateItem) error {
	tfPath := filepath.Join(cfg.WorkingDirectory, "main.tf")

	existingContent, err := os.ReadFile(tfPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read main.tf: %w", err)
	}

	if strings.Contains(string(existingContent), strings.TrimSpace(tmpl.Content)) {
		fmt.Println("Template already exists in main.tf, skipping duplicate injection.")
		return nil
	}

	file, err := os.OpenFile(tfPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open main.tf: %w", err)
	}
	defer file.Close()

	prefix := ""
	if len(existingContent) >= 2 && !(existingContent[len(existingContent)-2] == '\n' && existingContent[len(existingContent)-1] == '\n') {
		prefix = "\n"
	}

	_, err = file.WriteString(prefix + tmpl.Content + "\n\n")
	return err
}
