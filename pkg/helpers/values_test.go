package helpers

import (
	"os"
	"testing"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestGetDeletionPolicyDefault(t *testing.T) {
	policy := GetDeletionPolicy()
	assert.Equal(t, synv1alpha1.ArchivePolicy, policy)
}

func TestGetDeletionPolicyNonDefault(t *testing.T) {
	os.Setenv("DEFAULT_DELETION_POLICY", "Retain")
	policy := GetDeletionPolicy()
	assert.Equal(t, synv1alpha1.RetainPolicy, policy)
}

func TestRenderTemplateRawString(t *testing.T) {
	str, err := RenderTemplate("raw string", nil)
	assert.NoError(t, err)
	assert.Equal(t, "raw string", str)
}

func TestRenderTemplateData(t *testing.T) {
	str, err := RenderTemplate("{{ .Some }}/{{ .Data }}", struct {
		Some string
		Data string
	}{"some", "data"})
	assert.NoError(t, err)
	assert.Equal(t, "some/data", str)
}

func TestRenderTemplateSyntaxError(t *testing.T) {
	_, err := RenderTemplate("{{ }", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse")
}

func TestRenderTemplateDataError(t *testing.T) {
	_, err := RenderTemplate("{{ .none }}", "data")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "render")
}
