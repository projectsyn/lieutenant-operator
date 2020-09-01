package helpers

import (
	"os"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
)

const (
	DeleteProtectionAnnotation = "syn.tools/protected-delete"
)

func GetDeletionPolicy() synv1alpha1.DeletionPolicy {
	policy := synv1alpha1.DeletionPolicy(os.Getenv("DEFAULT_DELETION_POLICY"))
	switch policy {
	case synv1alpha1.ArchivePolicy, synv1alpha1.DeletePolicy, synv1alpha1.RetainPolicy:
		return policy
	default:
		return synv1alpha1.ArchivePolicy
	}
}
