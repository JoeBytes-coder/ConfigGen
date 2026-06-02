package generators

import (
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"text/template"

	"configgen/internal/domain"
	"configgen/internal/infrastructure/generators/templates"
)

func init() {
	Register("kustomize", &KustomizeGenerator{})
}

type KustomizeGenerator struct{}

type KustomizeTemplateData struct {
	AppName   string
	Image     string
	Tag       string
	Resources []string
	EnvLines  []string
}

func (g *KustomizeGenerator) Generate(req domain.ConfigRequest) (string, error) {
	if len(req.K8sResource) == 0 {
		return "", fmt.Errorf("at least one k8s resource type is required for kustomize")
	}

	resourceFileNames := make([]string, 0, len(req.K8sResource))
	resourceYamls := make([]string, 0, len(req.K8sResource))
	resourceSet := make(map[string]bool)

	for _, resource := range req.K8sResource {
		key := strings.ToLower(strings.TrimSpace(resource))
		resourceSet[key] = true
		yamlFile := key + ".yaml"
		resourceFileNames = append(resourceFileNames, yamlFile)

		config, err := renderK8sTemplate(key, req, req.K8sResource)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return "", fmt.Errorf("unsupported k8s resource type: %s", resource)
			}
			return "", err
		}
		resourceYamls = append(resourceYamls, config)
	}

	envLines := []string{}
	envKeys := make([]string, 0, len(req.Env))
	for k := range req.Env {
		envKeys = append(envKeys, k)
	}
	sort.Strings(envKeys)
	for _, k := range envKeys {
		envLines = append(envLines, fmt.Sprintf("%s=%s", k, req.Env[k]))
	}

	tpl, err := template.ParseFS(templates.KustomizeFS, "kustomize/kustomization.yaml.tmpl")
	if err != nil {
		return "", fmt.Errorf("failed to load kustomize template: %w", err)
	}

	data := KustomizeTemplateData{
		AppName:   req.AppName,
		Image:     req.Image,
		Tag:       req.Tag,
		Resources: resourceFileNames,
		EnvLines:  envLines,
	}

	var kustomizationSb strings.Builder
	if err := tpl.Execute(&kustomizationSb, data); err != nil {
		return "", fmt.Errorf("failed to execute kustomize template: %w", err)
	}

	parts := make([]string, 0, 1+len(resourceYamls))
	parts = append(parts, strings.TrimSpace(kustomizationSb.String()))
	for _, yml := range resourceYamls {
		parts = append(parts, yml)
	}

	return strings.Join(parts, "\n---\n"), nil
}
