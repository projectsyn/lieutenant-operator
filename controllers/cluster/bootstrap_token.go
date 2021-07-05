package cluster

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/pipeline"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func setBootstrapToken(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	instance, ok := obj.(*synv1alpha1.Cluster)
	if !ok {
		return pipeline.Result{Err: fmt.Errorf("%s is not a cluster object", obj.GetName())}
	}

	if instance.Status.BootstrapToken == nil {
		data.Log.Info("Adding status to Cluster object")
		err := newClusterStatus(instance)
		if err != nil {
			return pipeline.Result{Err: err}
		}
	}

	if time.Now().After(instance.Status.BootstrapToken.ValidUntil.Time) {
		instance.Status.BootstrapToken.TokenValid = false
	}

	return pipeline.Result{}
}

// newClusterStatus will create a default lifetime of 24h if it wasn't set in the object.
func newClusterStatus(cluster *synv1alpha1.Cluster) error {
	parseTime := "24h"
	if cluster.Spec.TokenLifeTime != "" {
		parseTime = cluster.Spec.TokenLifeTime
	}

	duration, err := time.ParseDuration(parseTime)
	if err != nil {
		return err
	}

	validUntil := time.Now().Add(duration)

	token, err := generateToken()
	if err != nil {
		return err
	}

	cluster.Status.BootstrapToken = &synv1alpha1.BootstrapToken{
		Token:      token,
		ValidUntil: metav1.NewTime(validUntil),
		TokenValid: true,
	}
	return nil
}

func generateToken() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), err
}
