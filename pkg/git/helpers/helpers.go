// Package helpers contains helper functions for the various git manipulations that can be re-used.
package helpers

import (
	"reflect"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
)

// CompareKey compares keySetA against keySetB, keySetA has the higher precedence. All keys that are in both sets, but differ,
// will be returned with the value of the set A instance. Also all keys not present in B but in A will also be added to the
// diff.
// This can be used to find out which keys have to be updated if keySetA is the local set of keys. It can also be used to
// find out which keys to delete on the repository if keySetA is the remote set of keys.
func CompareKeys(keySetA, keySetB map[string]synv1alpha1.DeployKey) map[string]synv1alpha1.DeployKey {
	deltaKeys := make(map[string]synv1alpha1.DeployKey)
	for k, v := range keySetA {
		if !reflect.DeepEqual(keySetB[k], v) {
			deltaKeys[k] = v
		}
	}
	return deltaKeys
}
