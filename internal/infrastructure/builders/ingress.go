package builders

import (
	"fmt"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"configgen/internal/domain"
)

func init() {
	register("Ingress", buildIngress)
	register("IngressClass", buildIngressClass)
}

func buildIngress(app domain.AppContext, n domain.Node) (any, error) {
	serviceRef, ok := n.Refs["service"]
	if !ok {
		return nil, fmt.Errorf("Ingress 需要 Service 引用")
	}
	if len(app.Ports) == 0 {
		return nil, fmt.Errorf("Ingress 需要至少一个端口")
	}
	pathType := networkingv1.PathTypePrefix
	rule := networkingv1.IngressRule{
		IngressRuleValue: networkingv1.IngressRuleValue{
			HTTP: &networkingv1.HTTPIngressRuleValue{
				Paths: []networkingv1.HTTPIngressPath{{
					Path:     "/",
					PathType: &pathType,
					Backend: networkingv1.IngressBackend{
						Service: &networkingv1.IngressServiceBackend{
							Name: serviceRef.Name,
							Port: networkingv1.ServiceBackendPort{Number: int32(app.Ports[0].Service)},
						},
					},
				}},
			},
		},
	}
	if app.Host != "" {
		rule.Host = app.Host
	}
	spec := networkingv1.IngressSpec{Rules: []networkingv1.IngressRule{rule}}
	if ref, ok := n.Refs["ingressclass"]; ok {
		name := ref.Name
		spec.IngressClassName = &name
	}
	if app.Host != "" && app.TLSSecret != "" {
		spec.TLS = []networkingv1.IngressTLS{{Hosts: []string{app.Host}, SecretName: app.TLSSecret}}
	}
	return &networkingv1.Ingress{
		TypeMeta:   metav1.TypeMeta{APIVersion: "networking.k8s.io/v1", Kind: "Ingress"},
		ObjectMeta: objectMeta(app, n),
		Spec:       spec,
	}, nil
}

func buildIngressClass(app domain.AppContext, n domain.Node) (any, error) {
	return &networkingv1.IngressClass{
		TypeMeta:   metav1.TypeMeta{APIVersion: "networking.k8s.io/v1", Kind: "IngressClass"},
		ObjectMeta: metav1.ObjectMeta{Name: n.Name, Labels: labels(app)},
		Spec:       networkingv1.IngressClassSpec{Controller: "k8s.io/ingress-nginx"},
	}, nil
}
