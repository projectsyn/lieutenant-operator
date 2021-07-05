package tenant

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func setManagedByLabel(obj metav1.Object) {
	obj.SetLabels(map[string]string{
		"app.kubernetes.io/managed-by": "lieutenant",
	})
}
