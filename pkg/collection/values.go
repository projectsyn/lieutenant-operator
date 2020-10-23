package collection

import (
	"bytes"
	"fmt"
	"text/template"

	corev1 "k8s.io/api/core/v1"
)

const (
	// DeleteProtectionAnnotation defines the delete protection annotation name
	DeleteProtectionAnnotation = "syn.tools/protected-delete"
)

type SecretSortList corev1.SecretList

func (s SecretSortList) Len() int      { return len(s.Items) }
func (s SecretSortList) Swap(i, j int) { s.Items[i], s.Items[j] = s.Items[j], s.Items[i] }

func (s SecretSortList) Less(i, j int) bool {

	if s.Items[i].CreationTimestamp.Equal(&s.Items[j].CreationTimestamp) {
		return s.Items[i].Name < s.Items[j].Name
	}

	return s.Items[i].CreationTimestamp.Before(&s.Items[j].CreationTimestamp)
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
