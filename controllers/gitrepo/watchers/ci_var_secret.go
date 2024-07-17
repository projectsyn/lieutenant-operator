package watchers

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
)

const (
	// GitRepoCIVariableValueFromSecretKeyRefNameIndex is the index name for the GitRepo objects that reference a secret by name.
	GitRepoCIVariableValueFromSecretKeyRefNameIndex = "spec.ciVariables.valueFrom.secretKeyRef.name"
)

// SecretGitRepoCIVariablesMapFunc returns a handler function that will return a list of reconcile.Requests for GitRepo objects
// that reference the secret in the given Secret object.
// It requires the field index GitRepoCIVariableValueFromSecretKeyRefNameIndex to be installed for the GitRepo objects.
func SecretGitRepoCIVariablesMapFunc(cli client.Client) func(ctx context.Context, o client.Object) []reconcile.Request {
	return func(ctx context.Context, o client.Object) []reconcile.Request {
		l := log.FromContext(ctx).WithName("SecretGitRepoCIVariablesMapFunc").WithValues("secret", o.GetName())

		secret := o.(*corev1.Secret)
		var gitRepos synv1alpha1.GitRepoList
		if err := cli.List(ctx, &gitRepos, client.MatchingFields{
			GitRepoCIVariableValueFromSecretKeyRefNameIndex: secret.Name,
		}, client.InNamespace(secret.GetNamespace())); err != nil {
			l.Error(err, "unable to list GitRepos")
			return []reconcile.Request{}
		}

		requests := make([]reconcile.Request, 0, len(gitRepos.Items))
		for _, gitRepo := range gitRepos.Items {
			requests = append(requests, reconcile.Request{
				NamespacedName: client.ObjectKey{
					Namespace: gitRepo.Namespace,
					Name:      gitRepo.Name,
				},
			})
		}

		return requests
	}
}

// GitRepoCIVariableValueFromSecretKeyRefNameIndexFunc is an index function for GitRepo objects.
// It indexes the names of the secrets that are referenced by the CIVariables of the GitRepo.
func GitRepoCIVariableValueFromSecretKeyRefNameIndexFunc(obj client.Object) []string {
	gitRepo := obj.(*synv1alpha1.GitRepo)
	values := make([]string, 0, len(gitRepo.Spec.CIVariables))
	for _, ciVariable := range gitRepo.Spec.CIVariables {
		if ciVariable.ValueFrom != nil &&
			ciVariable.ValueFrom.SecretKeyRef != nil &&
			ciVariable.ValueFrom.SecretKeyRef.Name != "" {
			values = append(values, ciVariable.ValueFrom.SecretKeyRef.Name)
		}
	}
	return values
}
