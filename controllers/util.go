package controllers

import (
	"slices"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
)

func envVarIndex(name string, list *[]synv1alpha1.EnvVar) int {
	for i, envvar := range *list {
		if envvar.Name == name {
			return i
		}
	}
	return -1
}

func updateEnvVarValue(name string, value string, envVars []synv1alpha1.EnvVar) ([]synv1alpha1.EnvVar, bool) {
	index := envVarIndex(name, &envVars)
	changed := false
	if index < 0 {
		changed = true
		envVars = append(envVars, synv1alpha1.EnvVar{
			GitlabOptions: synv1alpha1.EnvVarGitlabOptions{
				Raw: true,
			},
			Name:  name,
			Value: value,
		})
	} else if envVars[index].Value != value {
		changed = true
		envVars[index].Value = value
	}
	return envVars, changed
}
func updateEnvVarValueFrom(name string, secret string, key string, protected bool, envVars []synv1alpha1.EnvVar) ([]synv1alpha1.EnvVar, bool) {
	index := envVarIndex(name, &envVars)
	changed := false
	if index < 0 {
		changed = true
		envVars = append(envVars, synv1alpha1.EnvVar{
			Name: name,
			GitlabOptions: synv1alpha1.EnvVarGitlabOptions{
				Masked:    true,
				Raw:       true,
				Protected: protected,
			},
			ValueFrom: &synv1alpha1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{
						Name: secret,
					},
					Key: key,
				},
			},
		})
	} else if envVars[index].ValueFrom.SecretKeyRef.Name != secret || envVars[index].ValueFrom.SecretKeyRef.Key != key {
		changed = true
		envVars[index].ValueFrom = &synv1alpha1.EnvVarSource{
			SecretKeyRef: &v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{
					Name: secret,
				},
				Key: key,
			},
		}
	}
	return envVars, changed
}

func removeEnvVar(name string, envVars []synv1alpha1.EnvVar) ([]synv1alpha1.EnvVar, bool) {
	index := envVarIndex(name, &envVars)
	if index >= 0 {
		return slices.Delete(envVars, index, index+1), true

	}
	return envVars, false
}
