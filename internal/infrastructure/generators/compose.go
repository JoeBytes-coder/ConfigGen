package generators

import (
	"fmt"
	"sort"
	"strings"
	"text/template"

	"configgen/internal/domain"
	"configgen/internal/infrastructure/generators/templates"
)

func init() {
	Register("compose", &ComposeGenerator{})
}

type ComposeGenerator struct{}

type ComposeTemplateData struct {
	AppName string
	Image   string
	Tag     string
	Port    int
	Env     []string
}

func (g *ComposeGenerator) Generate(req domain.ConfigRequest) (string, error) {
	envLines := []string{}
	envKeys := make([]string, 0, len(req.Env))
	for k := range req.Env {
		envKeys = append(envKeys, k)
	}
	sort.Strings(envKeys)
	for _, k := range envKeys {
		envLines = append(envLines, fmt.Sprintf("      - %s=%s", k, req.Env[k]))
	}
	if len(envLines) == 0 {
		envLines = append(envLines, "      # no environment variables configured")
	}

	data := ComposeTemplateData{
		AppName: req.AppName,
		Image:   req.Image,
		Tag:     req.Tag,
		Port:    req.Port,
		Env:     envLines,
	}

	tpl, err := template.ParseFS(templates.ComposeFS, "compose/docker-compose.yaml.tmpl")
	if err != nil {
		return "", fmt.Errorf("failed to load compose template: %w", err)
	}

	var sb strings.Builder
	if err := tpl.Execute(&sb, data); err != nil {
		return "", fmt.Errorf("failed to execute compose template: %w", err)
	}

	return strings.TrimSpace(sb.String()), nil
}
