package helpers

import (
	"os"
	"testing"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/stretchr/testify/assert"
	"k8s.io/utils/pointer"
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

func TestSetTemplateIfEmptyRawString(t *testing.T) {
	testStruct := struct {
		some  string
		field *string
	}{
		"",
		pointer.StringPtr(""),
	}
	err := SetTemplateIfEmpty(&testStruct.some, "test", nil)
	assert.NoError(t, err)
	assert.Equal(t, "test", testStruct.some)

	err = SetTemplateIfEmpty(testStruct.field, "other", nil)
	assert.NoError(t, err)
	assert.Equal(t, "other", *testStruct.field)
}

func TestSetTemplateIfEmptyNil(t *testing.T) {
	err := SetTemplateIfEmpty(nil, "", nil)
	assert.Error(t, err)
}

func TestSetTemplateIfEmptyTemplate(t *testing.T) {
	str := pointer.StringPtr("")

	value := "some data"
	err := SetTemplateIfEmpty(str, "{{ .Data }}", struct{ Data string }{value})
	assert.NoError(t, err)
	assert.Equal(t, value, *str)
}

func TestSetTemplateIfNotEmpty(t *testing.T) {
	str := pointer.StringPtr("test")
	err := SetTemplateIfEmpty(str, "data", nil)
	assert.NoError(t, err)
	assert.Equal(t, "test", *str)
}

func TestSetTemplateIfEmptyTemplateError(t *testing.T) {
	str := pointer.StringPtr("")

	err := SetTemplateIfEmpty(str, "{{", nil)
	assert.Error(t, err)
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
