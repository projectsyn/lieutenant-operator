package role

import (
	"fmt"
	"reflect"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
)

func AddResourceNames(role *rbacv1.Role, name string) bool {
	i := EnsureRules(role)

	c, _ := contains(role.Rules[i].ResourceNames, name)
	if c {
		return false
	}

	role.Rules[i].ResourceNames = append(role.Rules[i].ResourceNames, name)

	return true
}

func RemoveResourceNames(role *rbacv1.Role, name string) bool {
	i, err := getRuleIndex(role)
	if err != nil {
		return false
	}

	names := role.Rules[i].ResourceNames

	c, j := contains(names, name)
	if !c {
		return false
	}

	// Remove element at position `i` assuming order is not relevant.
	names[j] = names[len(names)-1]
	role.Rules[i].ResourceNames = names[:len(names)-1]

	return true
}

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
		matchTenants, _ := contains(rule.Resources, "tenants")
		matchClusters, _ := contains(rule.Resources, "clusters")
		if matchGroups && matchVerbs && matchTenants && matchClusters {
			return i, nil
		}
	}
	return 0, fmt.Errorf("no matching rule")
}

func contains(items []string, value string) (bool, int) {
	for i, item := range items {
		if item == value {
			return true, i
		}
	}

	return false, 0
}
