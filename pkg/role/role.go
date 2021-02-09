package role

import (
	"fmt"
	"reflect"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
)

func EnsureRules(role *rbacv1.Role) int {
	if role.Rules == nil {
		role.Rules = []rbacv1.PolicyRule{}
	}
	i, err := getRuleIndex(role)
	if err != nil {
		rule := rbacv1.PolicyRule{
			APIGroups: []string{synv1alpha1.SchemeGroupVersion.Group},
			Verbs:     []string{"get"},
			Resources: []string{"tenants", "clusters"},
		}
		role.Rules = append(role.Rules, rule)
		i = len(role.Rules) - 1
	}

	return i
}

func getRuleIndex(role *rbacv1.Role) (int, error) {
	for i, rule := range role.Rules {
		matchGroups := reflect.DeepEqual(rule.APIGroups, []string{synv1alpha1.SchemeGroupVersion.Group})
		matchVerbs := reflect.DeepEqual(rule.Verbs, []string{"get"})
		matchResources := contains(rule.Resources, "tenants") && contains(rule.Resources, "clusters")
		if matchGroups && matchVerbs && matchResources {
			return i, nil
		}
	}
	return 0, fmt.Errorf("no matching rule")
}

func contains(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}

	return false
}
