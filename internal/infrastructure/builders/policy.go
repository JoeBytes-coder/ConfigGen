package builders

import (
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"configgen/internal/domain"
)

func init() {
	register("PodDisruptionBudget", buildPDB)
	register("ResourceQuota", buildResourceQuota)
	register("LimitRange", buildLimitRange)
}

func buildPDB(app domain.AppContext, n domain.Node) (any, error) {
	spec := policyv1.PodDisruptionBudgetSpec{Selector: &metav1.LabelSelector{MatchLabels: selectorLabels(app)}}
	if app.Replicas > 1 {
		min := intstr.FromInt32(1)
		spec.MinAvailable = &min
	} else {
		max := intstr.FromInt32(1)
		spec.MaxUnavailable = &max
	}
	return &policyv1.PodDisruptionBudget{
		TypeMeta:   metav1.TypeMeta{APIVersion: "policy/v1", Kind: "PodDisruptionBudget"},
		ObjectMeta: objectMeta(app, n),
		Spec:       spec,
	}, nil
}

func buildResourceQuota(app domain.AppContext, n domain.Node) (any, error) {
	return &corev1.ResourceQuota{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "ResourceQuota"},
		ObjectMeta: objectMeta(app, n),
		Spec: corev1.ResourceQuotaSpec{Hard: corev1.ResourceList{
			corev1.ResourceRequestsCPU:    resource.MustParse("4"),
			corev1.ResourceRequestsMemory: resource.MustParse("8Gi"),
			corev1.ResourceLimitsCPU:      resource.MustParse("8"),
			corev1.ResourceLimitsMemory:   resource.MustParse("16Gi"),
		}},
	}, nil
}

func buildLimitRange(app domain.AppContext, n domain.Node) (any, error) {
	return &corev1.LimitRange{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "LimitRange"},
		ObjectMeta: objectMeta(app, n),
		Spec: corev1.LimitRangeSpec{Limits: []corev1.LimitRangeItem{{
			Type: corev1.LimitTypeContainer,
			Default: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("512Mi"),
			},
			DefaultRequest: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("128Mi"),
			},
		}}},
	}, nil
}
