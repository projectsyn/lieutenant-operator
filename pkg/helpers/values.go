package helpers

import (
	"bytes"
	"fmt"
	"os"
	"text/template"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
)

const (
	// DeleteProtectionAnnotation defines the delete protection annotation name
	DeleteProtectionAnnotation = "syn.tools/protected-delete"
)

// GetDeletionPolicy gets the configured default deletion policy
func GetDeletionPolicy() synv1alpha1.DeletionPolicy {
	policy := synv1alpha1.DeletionPolicy(os.Getenv("DEFAULT_DELETION_POLICY"))
	switch policy {
	case synv1alpha1.ArchivePolicy, synv1alpha1.DeletePolicy, synv1alpha1.RetainPolicy:
		return policy
	default:
		return synv1alpha1.ArchivePolicy
	}
}

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

// SetTemplateIfEmpty renders the given template with data to the target field if it is empty
func SetTemplateIfEmpty(field *string, template string, data interface{}) error {
	if field == nil {
		return fmt.Errorf("Field may not be nil")
	}
	if len(*field) > 0 {
		return nil
	}
	value, err := RenderTemplate(template, data)
	if err != nil {
		return err
	}
	*field = value

	return nil
}
