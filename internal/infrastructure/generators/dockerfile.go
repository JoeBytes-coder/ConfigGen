package generators

import (
	"fmt"
	"sort"
	"strings"
	"text/template"

	"configgen/internal/domain"
	"configgen/internal/infrastructure/generators/templates"
)

type DockerfileGenerator struct{}

type DockerfileTemplateData struct {
	BaseImage string
	WorkDir   string
	AppName   string
	Image     string
	Tag       string
	Port      int
	Env       []string
	Cmd       []string
}

func (g *DockerfileGenerator) Generate(req domain.ConfigRequest) (string, error) {
	envLines := []string{}
	envKeys := make([]string, 0, len(req.Env))
	for k := range req.Env {
		envKeys = append(envKeys, k)
	}
	sort.Strings(envKeys)
	for _, k := range envKeys {
		envLines = append(envLines, fmt.Sprintf("ENV %s=%s", k, req.Env[k]))
	}

	baseImage := req.DockerfileBaseImage
	if baseImage == "" {
		baseImage = fmt.Sprintf("%s:%s", req.Image, req.Tag)
	}

	workDir := req.DockerfileWorkDir
	if workDir == "" {
		workDir = "/app"
	}

	cmd := req.DockerfileCmd
	if len(cmd) == 0 {
		cmd = []string{"./" + req.AppName}
	}

	data := DockerfileTemplateData{
		BaseImage: baseImage,
		WorkDir:   workDir,
		AppName:   req.AppName,
		Image:     req.Image,
		Tag:       req.Tag,
		Port:      req.Port,
		Env:       envLines,
		Cmd:       cmd,
	}

	tpl, err := template.ParseFS(templates.DockerfileFS, "dockerfile/Dockerfile.tmpl")
	if err != nil {
		return "", fmt.Errorf("failed to load dockerfile template: %w", err)
	}

	var buf strings.Builder
	if err := tpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute dockerfile template: %w", err)
	}

	return buf.String(), nil
}
