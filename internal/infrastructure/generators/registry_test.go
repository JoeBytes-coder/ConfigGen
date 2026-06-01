package generators

import (
	"strings"
	"testing"

	"configgen/internal/domain"
)

func TestGenerateConfig(t *testing.T) {
	t.Run("unknown type returns error", func(t *testing.T) {
		req := domain.ConfigRequest{
			Type:    "unknown",
			AppName: "test",
			Image:   "nginx",
			Tag:     "latest",
			Port:    80,
		}
		_, err := GenerateConfig(req)
		if err == nil {
			t.Fatal("expected error for unknown type")
		}
	})

	t.Run("compose type succeeds", func(t *testing.T) {
		req := domain.ConfigRequest{
			Type:    "compose",
			AppName: "compose-app",
			Image:   "nginx",
			Tag:     "latest",
			Port:    80,
		}
		result, err := GenerateConfig(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Type != "compose" {
			t.Errorf("expected type compose, got %s", result.Type)
		}
		if result.Config == "" {
			t.Error("expected non-empty config")
		}
	})

	t.Run("k8s type succeeds", func(t *testing.T) {
		req := domain.ConfigRequest{
			Type:        "k8s",
			AppName:     "k8s-app",
			Image:       "nginx",
			Tag:         "latest",
			Port:        80,
			K8sResource: []string{"Namespace"},
		}
		result, err := GenerateConfig(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Type != "k8s" {
			t.Errorf("expected type k8s, got %s", result.Type)
		}
	})

	t.Run("dockerfile type succeeds", func(t *testing.T) {
		req := domain.ConfigRequest{
			Type:    "dockerfile",
			AppName: "df-app",
			Image:   "golang",
			Tag:     "1.21",
			Port:    8080,
		}
		result, err := GenerateConfig(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Type != "dockerfile" {
			t.Errorf("expected type dockerfile, got %s", result.Type)
		}
	})

	t.Run("kustomize type succeeds", func(t *testing.T) {
		req := domain.ConfigRequest{
			Type:        "kustomize",
			AppName:     "kustomize-app",
			Image:       "nginx",
			Tag:         "latest",
			Port:        80,
			K8sResource: []string{"Deployment", "Service"},
		}
		result, err := GenerateConfig(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Type != "kustomize" {
			t.Errorf("expected type kustomize, got %s", result.Type)
		}
		if !strings.Contains(result.Config, "apiVersion: kustomize.config.k8s.io") {
			t.Error("expected kustomization.yaml content")
		}
	})
}
