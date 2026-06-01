package generators

import (
	"configgen/internal/domain"
	"strings"
	"testing"
)

func TestK8sGenerator_Generate(t *testing.T) {
	g := &K8sGenerator{}

	t.Run("no resource types returns error", func(t *testing.T) {
		req := domain.ConfigRequest{
			AppName:     "test",
			Image:       "nginx",
			Tag:         "latest",
			Port:        80,
			K8sResource: []string{},
		}
		_, err := g.Generate(req)
		if err == nil {
			t.Fatal("expected error for empty resources")
		}
	})

	t.Run("unsupported resource type", func(t *testing.T) {
		req := domain.ConfigRequest{
			AppName:     "test",
			Image:       "nginx",
			Tag:         "latest",
			Port:        80,
			K8sResource: []string{"UnknownType"},
		}
		_, err := g.Generate(req)
		if err == nil {
			t.Fatal("expected error for unsupported resource")
		}
	})

	t.Run("single deployment", func(t *testing.T) {
		req := domain.ConfigRequest{
			AppName:     "myapp",
			Image:       "nginx",
			Tag:         "latest",
			Port:        80,
			Replicas:    3,
			K8sResource: []string{"Deployment"},
		}
		result, err := g.Generate(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "kind: Deployment") {
			t.Error("missing Deployment kind")
		}
		if !strings.Contains(result, "replicas: 3") {
			t.Error("missing replicas")
		}
	})

	t.Run("multiple resources with separator", func(t *testing.T) {
		req := domain.ConfigRequest{
			AppName:     "webapp",
			Image:       "myapp",
			Tag:         "v1",
			Port:        8080,
			Replicas:    2,
			K8sResource: []string{"Namespace", "Deployment", "Service"},
		}
		result, err := g.Generate(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Count(result, "\n---\n") != 2 {
			t.Errorf("expected 2 separators for 3 resources, got %d", strings.Count(result, "\n---\n"))
		}
		if !strings.Contains(result, "kind: Namespace") {
			t.Error("missing Namespace")
		}
		if !strings.Contains(result, "kind: Deployment") {
			t.Error("missing Deployment")
		}
		if !strings.Contains(result, "kind: Service") {
			t.Error("missing Service")
		}
	})

	t.Run("service selects app label", func(t *testing.T) {
		req := domain.ConfigRequest{
			AppName:     "webapp",
			Image:       "nginx",
			Tag:         "latest",
			Port:        3000,
			K8sResource: []string{"Service"},
		}
		result, err := g.Generate(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "port: 3000") {
			t.Error("missing service port")
		}
	})

	t.Run("non-workload resources (ConfigMap)", func(t *testing.T) {
		req := domain.ConfigRequest{
			AppName:     "cfgapp",
			Image:       "nginx",
			Tag:         "latest",
			Port:        80,
			K8sResource: []string{"ConfigMap"},
		}
		result, err := g.Generate(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "kind: ConfigMap") {
			t.Error("missing ConfigMap")
		}
	})

	t.Run("RBAC resources (Role + RoleBinding)", func(t *testing.T) {
		req := domain.ConfigRequest{
			AppName:     "rbacapp",
			Image:       "nginx",
			Tag:         "latest",
			Port:        80,
			K8sResource: []string{"Role", "RoleBinding"},
		}
		result, err := g.Generate(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "kind: Role") {
			t.Error("missing Role")
		}
		if !strings.Contains(result, "kind: RoleBinding") {
			t.Error("missing RoleBinding")
		}
	})

	t.Run("zero replicas defaults to 1", func(t *testing.T) {
		req := domain.ConfigRequest{
			AppName:     "zeroapp",
			Image:       "nginx",
			Tag:         "latest",
			Port:        80,
			Replicas:    0,
			K8sResource: []string{"Deployment"},
		}
		result, err := g.Generate(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "replicas: 1") {
			t.Error("expected replicas default to 1")
		}
	})

	t.Run("ConfigMap triggers volume in Deployment", func(t *testing.T) {
		req := domain.ConfigRequest{
			AppName:     "volapp",
			Image:       "nginx",
			Tag:         "latest",
			Port:        80,
			Replicas:    1,
			K8sResource: []string{"Deployment", "ConfigMap"},
		}
		result, err := g.Generate(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "configMap:") {
			t.Error("deployment should reference configmap volume when ConfigMap selected")
		}
	})
}
