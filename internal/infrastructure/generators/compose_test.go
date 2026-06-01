package generators

import (
	"configgen/internal/domain"
	"strings"
	"testing"
)

func TestComposeGenerator_Generate(t *testing.T) {
	g := &ComposeGenerator{}

	t.Run("basic service", func(t *testing.T) {
		req := domain.ConfigRequest{
			AppName: "myapp",
			Image:   "nginx",
			Tag:     "latest",
			Port:    80,
		}
		result, err := g.Generate(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "version: \"3.8\"") {
			t.Error("missing version header")
		}
		if !strings.Contains(result, "myapp:") {
			t.Error("missing service name")
		}
		if !strings.Contains(result, "nginx:latest") {
			t.Error("missing image:tag")
		}
		if !strings.Contains(result, "\"80:80\"") {
			t.Error("missing port mapping")
		}
		if !strings.Contains(result, "restart: unless-stopped") {
			t.Error("missing restart policy")
		}
	})

	t.Run("with environment variables", func(t *testing.T) {
		req := domain.ConfigRequest{
			AppName: "webapp",
			Image:   "myapp",
			Tag:     "v1.0",
			Port:    3000,
			Env: map[string]string{
				"NODE_ENV": "production",
				"DEBUG":    "false",
			},
		}
		result, err := g.Generate(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "NODE_ENV=production") {
			t.Error("missing NODE_ENV env var")
		}
		if !strings.Contains(result, "DEBUG=false") {
			t.Error("missing DEBUG env var")
		}
	})

	t.Run("no duplicate restart lines", func(t *testing.T) {
		req := domain.ConfigRequest{
			AppName: "test",
			Image:   "alpine",
			Tag:     "3.18",
			Port:    8080,
		}
		result, err := g.Generate(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count := strings.Count(result, "restart: unless-stopped")
		if count != 1 {
			t.Errorf("expected 1 restart line, got %d", count)
		}
	})

	t.Run("no environment generates placeholder comment", func(t *testing.T) {
		req := domain.ConfigRequest{
			AppName: "noenv",
			Image:   "busybox",
			Tag:     "latest",
			Port:    80,
			Env:     nil,
		}
		result, err := g.Generate(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "no environment variables configured") {
			t.Error("missing placeholder comment for no env vars")
		}
	})
}
