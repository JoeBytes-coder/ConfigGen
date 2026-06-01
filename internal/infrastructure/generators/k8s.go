package generators

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"sync"
	"text/template"

	"configgen/internal/domain"
	"configgen/internal/infrastructure/generators/templates"
)

type K8sGenerator struct{}

type K8sTemplateData struct {
	AppName           string
	Image             string
	Tag               string
	Port              int
	Replicas          int
	HasConfigMap      bool
	HasService        bool
	HasServiceAccount bool
	HasRole           bool
	HasClusterRole    bool
}

var (
	k8sTemplateCache   = map[string]*template.Template{}
	k8sTemplateCacheMu sync.RWMutex
)

func loadK8sTemplate(kind string) (*template.Template, error) {
	key := strings.ToLower(strings.TrimSpace(kind))
	k8sTemplateCacheMu.RLock()
	tpl, ok := k8sTemplateCache[key]
	k8sTemplateCacheMu.RUnlock()
	if ok {
		return tpl, nil
	}

	filename := fmt.Sprintf("k8s/%s.yaml.tmpl", key)
	tpl, err := template.ParseFS(templates.K8sFS, filename)
	if err != nil {
		return nil, err
	}

	k8sTemplateCacheMu.Lock()
	defer k8sTemplateCacheMu.Unlock()
	if existing, ok := k8sTemplateCache[key]; ok {
		return existing, nil
	}
	k8sTemplateCache[key] = tpl

	return tpl, nil
}

func renderK8sTemplate(kind string, req domain.ConfigRequest, resources []string) (string, error) {
	tpl, err := loadK8sTemplate(kind)
	if err != nil {
		return "", err
	}

	replicas := req.Replicas
	if replicas <= 0 {
		replicas = 1
	}

	resourceSet := make(map[string]bool)
	for _, r := range resources {
		resourceSet[strings.ToLower(strings.TrimSpace(r))] = true
	}

	data := K8sTemplateData{
		AppName:           req.AppName,
		Image:             req.Image,
		Tag:               req.Tag,
		Port:              req.Port,
		Replicas:          replicas,
		HasConfigMap:      resourceSet["configmap"],
		HasService:        resourceSet["service"],
		HasServiceAccount: resourceSet["serviceaccount"],
		HasRole:           resourceSet["role"],
		HasClusterRole:    resourceSet["clusterrole"],
	}

	var sb strings.Builder
	if err := tpl.Execute(&sb, data); err != nil {
		return "", err
	}

	return strings.TrimSpace(sb.String()), nil
}

func (g *K8sGenerator) Generate(req domain.ConfigRequest) (string, error) {
	if len(req.K8sResource) == 0 {
		return "", fmt.Errorf("at least one k8s resource type is required")
	}

	var configs []string
	for _, resource := range req.K8sResource {
		key := strings.ToLower(strings.TrimSpace(resource))

		config, err := renderK8sTemplate(key, req, req.K8sResource)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return "", fmt.Errorf("unsupported k8s resource type: %s", resource)
			}
			return "", err
		}

		configs = append(configs, config)
	}

	return strings.Join(configs, "\n---\n"), nil
}
