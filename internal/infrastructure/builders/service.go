package builders

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"configgen/internal/domain"
)

func init() {
	register("Service", buildService)
}

func buildService(app domain.AppContext, n domain.Node) (any, error) {
	if len(app.Ports) == 0 {
		return nil, fmt.Errorf("Service 需要至少一个端口")
	}
	ports := make([]corev1.ServicePort, 0, len(app.Ports))
	for _, p := range app.Ports {
		ports = append(ports, corev1.ServicePort{
			Name:       p.Name,
			Protocol:   corev1.ProtocolTCP,
			Port:       int32(p.Service),
			TargetPort: intstr.FromInt32(int32(p.Container)),
		})
	}
	spec := corev1.ServiceSpec{Selector: selectorLabels(app), Ports: ports}
	if ref, ok := n.Refs["workload"]; ok && ref.Kind == "StatefulSet" {
		spec.ClusterIP = corev1.ClusterIPNone
	}
	return &corev1.Service{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Service"},
		ObjectMeta: objectMeta(app, n),
		Spec:       spec,
	}, nil
}
