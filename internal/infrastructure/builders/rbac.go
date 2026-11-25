package builders

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"configgen/internal/domain"
)

func init() {
	register("ServiceAccount", buildServiceAccount)
	register("Role", buildRole)
	register("ClusterRole", buildClusterRole)
	register("RoleBinding", buildRoleBinding)
	register("ClusterRoleBinding", buildClusterRoleBinding)
}

func podListRules() []rbacv1.PolicyRule {
	return []rbacv1.PolicyRule{{
		APIGroups: []string{""},
		Resources: []string{"pods"},
		Verbs:     []string{"get", "list", "watch"},
	}}
}

func buildServiceAccount(app domain.AppContext, n domain.Node) (any, error) {
	return &corev1.ServiceAccount{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "ServiceAccount"},
		ObjectMeta: objectMeta(app, n),
	}, nil
}

func buildRole(app domain.AppContext, n domain.Node) (any, error) {
	return &rbacv1.Role{
		TypeMeta:   metav1.TypeMeta{APIVersion: "rbac.authorization.k8s.io/v1", Kind: "Role"},
		ObjectMeta: objectMeta(app, n),
		Rules:      podListRules(),
	}, nil
}

func buildClusterRole(app domain.AppContext, n domain.Node) (any, error) {
	return &rbacv1.ClusterRole{
		TypeMeta:   metav1.TypeMeta{APIVersion: "rbac.authorization.k8s.io/v1", Kind: "ClusterRole"},
		ObjectMeta: metav1.ObjectMeta{Name: n.Name, Labels: labels(app)},
		Rules:      podListRules(),
	}, nil
}

func buildRoleBinding(app domain.AppContext, n domain.Node) (any, error) {
	sa, ok := n.Refs["sa"]
	if !ok {
		return nil, fmt.Errorf("RoleBinding 需要 ServiceAccount 引用")
	}
	roleRef, ok := n.Refs["role"]
	if !ok {
		roleRef, ok = n.Refs["clusterrole"]
	}
	if !ok {
		return nil, fmt.Errorf("RoleBinding 需要 Role 或 ClusterRole 引用")
	}
	return &rbacv1.RoleBinding{
		TypeMeta:   metav1.TypeMeta{APIVersion: "rbac.authorization.k8s.io/v1", Kind: "RoleBinding"},
		ObjectMeta: objectMeta(app, n),
		Subjects:   []rbacv1.Subject{{Kind: "ServiceAccount", Name: sa.Name, Namespace: n.Namespace}},
		RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: roleRef.Kind, Name: roleRef.Name},
	}, nil
}

func buildClusterRoleBinding(app domain.AppContext, n domain.Node) (any, error) {
	sa, ok := n.Refs["sa"]
	if !ok {
		return nil, fmt.Errorf("ClusterRoleBinding 需要 ServiceAccount 引用")
	}
	roleRef, ok := n.Refs["clusterrole"]
	if !ok {
		return nil, fmt.Errorf("ClusterRoleBinding 需要 ClusterRole 引用")
	}
	namespace := n.Namespace
	if namespace == "" {
		namespace = "default"
	}
	return &rbacv1.ClusterRoleBinding{
		TypeMeta:   metav1.TypeMeta{APIVersion: "rbac.authorization.k8s.io/v1", Kind: "ClusterRoleBinding"},
		ObjectMeta: metav1.ObjectMeta{Name: n.Name, Labels: labels(app)},
		Subjects:   []rbacv1.Subject{{Kind: "ServiceAccount", Name: sa.Name, Namespace: namespace}},
		RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: roleRef.Kind, Name: roleRef.Name},
	}, nil
}
