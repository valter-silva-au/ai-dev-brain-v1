package core

import (
	"bytes"
	"embed"
	"fmt"
	"path"
	"text/template"
)

// TemplateType represents the type of template to render
type TemplateType string

const (
	// TemplateTypeNotes represents a notes.md template
	TemplateTypeNotes TemplateType = "notes.md"
	// TemplateTypeDesign represents a design.md template
	TemplateTypeDesign TemplateType = "design.md"
	// TemplateTypeHandoff represents a handoff.md template
	TemplateTypeHandoff TemplateType = "handoff.md"
	// TemplateTypeStatus represents a status.yaml template
	TemplateTypeStatus TemplateType = "status.yaml"
	// TemplateTypeContext represents a context.md template
	TemplateTypeContext TemplateType = "context.md"
	// TemplateTypeTaskContext represents a task-context.md template for .claude/rules/
	TemplateTypeTaskContext TemplateType = "task-context.md"
)

// TemplateManager defines the interface for rendering templates
type TemplateManager interface {
	// Render renders a template with the given data
	Render(templateType TemplateType, data interface{}) (string, error)
	// RenderBytes renders a template with the given data and returns bytes
	RenderBytes(templateType TemplateType, data interface{}) ([]byte, error)
}

// EmbedTemplateManager implements TemplateManager using embedded templates
type EmbedTemplateManager struct {
	fs        embed.FS
	templates map[TemplateType]*template.Template
}

// NewEmbedTemplateManager creates a new template manager with embedded templates
// The fs parameter should be an embed.FS containing template files
func NewEmbedTemplateManager(fs embed.FS) (*EmbedTemplateManager, error) {
	tm := &EmbedTemplateManager{
		fs:        fs,
		templates: make(map[TemplateType]*template.Template),
	}

	// Pre-parse all templates
	templateTypes := []TemplateType{
		TemplateTypeNotes,
		TemplateTypeDesign,
		TemplateTypeHandoff,
		TemplateTypeStatus,
		TemplateTypeContext,
		TemplateTypeTaskContext,
	}

	for _, tt := range templateTypes {
		if err := tm.loadTemplate(tt); err != nil {
			return nil, fmt.Errorf("failed to load template %s: %w", tt, err)
		}
	}

	return tm, nil
}

// loadTemplate loads and parses a template file from the embedded filesystem
func (tm *EmbedTemplateManager) loadTemplate(templateType TemplateType) error {
	// Use path.Join (not filepath.Join) for embed.FS paths
	templatePath := path.Join(string(templateType))

	// Read template content from embedded filesystem
	content, err := tm.fs.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template file: %w", err)
	}

	// Parse template
	tmpl, err := template.New(string(templateType)).Parse(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	tm.templates[templateType] = tmpl
	return nil
}

// Render renders a template with the given data and returns the result as a string
func (tm *EmbedTemplateManager) Render(templateType TemplateType, data interface{}) (string, error) {
	bytes, err := tm.RenderBytes(templateType, data)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// RenderBytes renders a template with the given data and returns the result as bytes
func (tm *EmbedTemplateManager) RenderBytes(templateType TemplateType, data interface{}) ([]byte, error) {
	tmpl, ok := tm.templates[templateType]
	if !ok {
		return nil, fmt.Errorf("template %s not found", templateType)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}
