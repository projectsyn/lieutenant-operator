package pipeline

import (
	"os"
	"testing"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"gotest.tools/assert"
)

func TestGetDeletionPolicyDefault(t *testing.T) {
	policy := getDefaultDeletionPolicy()
	assert.Equal(t, synv1alpha1.ArchivePolicy, policy)
}

func TestGetDeletionPolicyNonDefault(t *testing.T) {
	os.Setenv("DEFAULT_DELETION_POLICY", "Retain")
	policy := getDefaultDeletionPolicy()
	assert.Equal(t, synv1alpha1.RetainPolicy, policy)
}
