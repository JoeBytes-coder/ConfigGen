package builders

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"configgen/internal/domain"
)

func init() {
	register("ConfigMap", buildConfigMap)
	register("Secret", buildSecret)
}

func buildConfigMap(app domain.AppContext, n domain.Node) (any, error) {
	data := make(map[string]string, len(app.Env))
	for k, v := range app.Env {
		data[k] = v
	}
	return &corev1.ConfigMap{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"},
		ObjectMeta: objectMeta(app, n),
		Data:       data,
	}, nil
}

func buildSecret(app domain.AppContext, n domain.Node) (any, error) {
	data := make(map[string]string, len(app.SecretEnv))
	for k, v := range app.SecretEnv {
		data[k] = v
	}
	return &corev1.Secret{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
		ObjectMeta: objectMeta(app, n),
		Type:       corev1.SecretTypeOpaque,
		StringData: data,
	}, nil
}
