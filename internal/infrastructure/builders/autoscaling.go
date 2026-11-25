package builders

import (
	"fmt"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"configgen/internal/domain"
)

func init() {
	register("HorizontalPodAutoscaler", buildHPA)
}

func buildHPA(app domain.AppContext, n domain.Node) (any, error) {
	ref, ok := n.Refs["workload"]
	if !ok {
		return nil, fmt.Errorf("HPA 需要可伸缩工作负载引用")
	}
	min := app.Replicas
	if min < 1 {
		min = 1
	}
	max := min * 2
	if max < 5 {
		max = 5
	}
	utilization := int32(70)
	apiVersion := "apps/v1"
	if spec, ok := domain.SpecByKind(ref.Kind); ok {
		apiVersion = spec.APIVersion
	}
	return &autoscalingv2.HorizontalPodAutoscaler{
		TypeMeta:   metav1.TypeMeta{APIVersion: "autoscaling/v2", Kind: "HorizontalPodAutoscaler"},
		ObjectMeta: objectMeta(app, n),
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{APIVersion: apiVersion, Kind: ref.Kind, Name: ref.Name},
			MinReplicas:    &min,
			MaxReplicas:    max,
			Metrics: []autoscalingv2.MetricSpec{{
				Type: autoscalingv2.ResourceMetricSourceType,
				Resource: &autoscalingv2.ResourceMetricSource{
					Name: corev1.ResourceCPU,
					Target: autoscalingv2.MetricTarget{
						Type:               autoscalingv2.UtilizationMetricType,
						AverageUtilization: &utilization,
					},
				},
			}},
		},
	}, nil
}
