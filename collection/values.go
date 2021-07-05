package collection

import (
	corev1 "k8s.io/api/core/v1"
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
