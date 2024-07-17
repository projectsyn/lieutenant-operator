package watchers_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/controllers/gitrepo/watchers"
)

func TestIndexAndMapFunc(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, synv1alpha1.AddToScheme(scheme))

	defaultNs := "test-namespace"

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "important-ci-stuff",
			Namespace: defaultNs,
		},
		Data: map[string][]byte{
			"key": []byte("value"),
		},
	}
	secret2 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "important-ci-stuff-too",
			Namespace: defaultNs,
		},
		Data: map[string][]byte{
			"key": []byte("value"),
		},
	}

	repoWithSecretRef := &synv1alpha1.GitRepo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "repo-with-secret-ref",
			Namespace: secret.GetNamespace(),
		},
		Spec: synv1alpha1.GitRepoSpec{
			GitRepoTemplate: synv1alpha1.GitRepoTemplate{
				CIVariables: []synv1alpha1.EnvVar{
					{
						Name:  "KEY",
						Value: "value",
					},
					{
						Name: "SECRET",
						ValueFrom: &synv1alpha1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: secret.Name,
								},
								Key: "key",
							},
						},
					},
					{
						Name: "SECRET2",
						ValueFrom: &synv1alpha1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: secret2.Name,
								},
								Key: "key",
							},
						},
					},
				},
			},
		},
	}
	repoWithOtherSecretRef := &synv1alpha1.GitRepo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "repo-with-other-secret-ref",
			Namespace: secret.GetNamespace(),
		},
		Spec: synv1alpha1.GitRepoSpec{
			GitRepoTemplate: synv1alpha1.GitRepoTemplate{
				CIVariables: []synv1alpha1.EnvVar{
					{
						Name:  "KEY",
						Value: "value",
					},
					{
						Name: "SECRET",
						ValueFrom: &synv1alpha1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "other-secret",
								},
								Key: "key",
							},
						},
					},
				},
			},
		},
	}
	repoWithSecretRefInOtherNs := &synv1alpha1.GitRepo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "repo-with-secret-ref-in-other-ns",
			Namespace: "other-namespace",
		},
		Spec: synv1alpha1.GitRepoSpec{
			GitRepoTemplate: synv1alpha1.GitRepoTemplate{
				CIVariables: []synv1alpha1.EnvVar{
					{
						Name:  "KEY",
						Value: "value",
					},
					{
						Name: "SECRET",
						ValueFrom: &synv1alpha1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "other-secret",
								},
								Key: "key",
							},
						},
					},
				},
			},
		},
	}
	repoWithoutRef := &synv1alpha1.GitRepo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "repo-without-ref",
			Namespace: defaultNs,
		},
		Spec: synv1alpha1.GitRepoSpec{
			GitRepoTemplate: synv1alpha1.GitRepoTemplate{},
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(secret, secret2, repoWithSecretRef, repoWithOtherSecretRef, repoWithSecretRefInOtherNs, repoWithoutRef).
		WithIndex(&synv1alpha1.GitRepo{}, watchers.GitRepoCIVariableValueFromSecretKeyRefNameIndex, watchers.GitRepoCIVariableValueFromSecretKeyRefNameIndexFunc).
		Build()

	requests := watchers.SecretGitRepoCIVariablesMapFunc(c)(context.Background(), secret)
	require.Len(t, requests, 1)
	require.Equal(t, repoWithSecretRef.Name, requests[0].Name)
	requests = watchers.SecretGitRepoCIVariablesMapFunc(c)(context.Background(), secret2)
	require.Len(t, requests, 1)
	require.Equal(t, repoWithSecretRef.Name, requests[0].Name)
}
