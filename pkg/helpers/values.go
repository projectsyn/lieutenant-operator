package helpers

import (
	"bytes"
	"fmt"
	"text/template"
)

const (
	// DeleteProtectionAnnotation defines the delete protection annotation name
	DeleteProtectionAnnotation = "syn.tools/protected-delete"
)

// RenderTemplate renders a given template with the given data
func RenderTemplate(tmpl string, data interface{}) (string, error) {
	tmp, err := template.New("template").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("Could not parse template: %w", err)
	}
	buf := new(bytes.Buffer)
	if err := tmp.Execute(buf, data); err != nil {
		return "", fmt.Errorf("Could not render template: %w", err)
	}
	return buf.String(), nil
}
