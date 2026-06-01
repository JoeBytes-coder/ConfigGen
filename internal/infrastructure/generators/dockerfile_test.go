package generators

import (
	"configgen/internal/domain"
	"strings"
	"testing"
)

func TestDockerfileGenerator_Generate(t *testing.T) {
	g := &DockerfileGenerator{}

	t.Run("default values", func(t *testing.T) {
		req := domain.ConfigRequest{
			AppName: "myapp",
			Image:   "golang",
			Tag:     "1.21",
			Port:    8080,
		}
		result, err := g.Generate(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "FROM golang:1.21") {
			t.Error("missing FROM with default base image")
		}
		if !strings.Contains(result, "WORKDIR /app") {
			t.Error("missing default WORKDIR")
		}
		if !strings.Contains(result, "EXPOSE 8080") {
			t.Error("missing EXPOSE")
		}
		if !strings.Contains(result, "./myapp") {
			t.Error("missing default CMD")
		}
	})

	t.Run("custom base image and workdir", func(t *testing.T) {
		req := domain.ConfigRequest{
			AppName:             "webserver",
			Image:               "nginx",
			Tag:                 "alpine",
			Port:                80,
			DockerfileBaseImage: "alpine:3.18",
			DockerfileWorkDir:   "/opt/app",
		}
		result, err := g.Generate(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "FROM alpine:3.18") {
			t.Error("missing custom base image")
		}
		if !strings.Contains(result, "WORKDIR /opt/app") {
			t.Error("missing custom workdir")
		}
	})

	t.Run("custom command", func(t *testing.T) {
		req := domain.ConfigRequest{
			AppName:       "server",
			Image:         "golang",
			Tag:           "1.21",
			Port:          9090,
			DockerfileCmd: []string{"/bin/server", "--port", "9090"},
		}
		result, err := g.Generate(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "/bin/server") {
			t.Error("missing custom CMD")
		}
		if !strings.Contains(result, "--port") {
			t.Error("missing CMD args")
		}
	})

	t.Run("with environment variables", func(t *testing.T) {
		req := domain.ConfigRequest{
			AppName: "envapp",
			Image:   "node",
			Tag:     "18",
			Port:    3000,
			Env: map[string]string{
				"NODE_ENV": "production",
			},
		}
		result, err := g.Generate(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "ENV NODE_ENV=production") {
			t.Error("missing ENV directive")
		}
	})
}
