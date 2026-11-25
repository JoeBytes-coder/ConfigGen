package builders

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"configgen/internal/domain"
)

func init() {
	register("NetworkPolicy", buildNetworkPolicy)
	register("EndpointSlice", buildEndpointSlice)
}

func buildNetworkPolicy(app domain.AppContext, n domain.Node) (any, error) {
	empty := &metav1.LabelSelector{}
	udp := corev1.ProtocolUDP
	tcp := corev1.ProtocolTCP
	dnsPort := intstr.FromInt32(53)
	dnsNamespace := &metav1.LabelSelector{MatchLabels: map[string]string{"kubernetes.io/metadata.name": "kube-system"}}
	return &networkingv1.NetworkPolicy{
		TypeMeta:   metav1.TypeMeta{APIVersion: "networking.k8s.io/v1", Kind: "NetworkPolicy"},
		ObjectMeta: objectMeta(app, n),
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{MatchLabels: selectorLabels(app)},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{From: []networkingv1.NetworkPolicyPeer{{PodSelector: empty}}},
			},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{To: []networkingv1.NetworkPolicyPeer{{PodSelector: empty}}},
				{
					To:    []networkingv1.NetworkPolicyPeer{{NamespaceSelector: dnsNamespace}},
					Ports: []networkingv1.NetworkPolicyPort{{Protocol: &udp, Port: &dnsPort}, {Protocol: &tcp, Port: &dnsPort}},
				},
			},
		},
	}, nil
}

func buildEndpointSlice(app domain.AppContext, n domain.Node) (any, error) {
	ref, ok := n.Refs["service"]
	if !ok {
		return nil, fmt.Errorf("EndpointSlice 需要 Service 引用")
	}
	meta := objectMeta(app, n)
	meta.Labels["kubernetes.io/service-name"] = ref.Name
	ports := make([]discoveryv1.EndpointPort, 0, len(app.Ports))
	for _, p := range app.Ports {
		name := p.Name
		port := int32(p.Container)
		protocol := corev1.ProtocolTCP
		ports = append(ports, discoveryv1.EndpointPort{Name: &name, Port: &port, Protocol: &protocol})
	}
	return &discoveryv1.EndpointSlice{
		TypeMeta:    metav1.TypeMeta{APIVersion: "discovery.k8s.io/v1", Kind: "EndpointSlice"},
		ObjectMeta:  meta,
		AddressType: discoveryv1.AddressTypeIPv4,
		Endpoints:   []discoveryv1.Endpoint{},
		Ports:       ports,
	}, nil
}
