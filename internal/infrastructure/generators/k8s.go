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

func init() {
	Register("k8s", &K8sGenerator{})
}

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

// resourceName returns the conventional K8s resource name for a given kind and app name.
func resourceName(kind, appName string) string {
	switch kind {
	case "Deployment":
		return appName + "-deployment"
	case "Service":
		return appName + "-service"
	case "ConfigMap":
		return appName + "-configmap"
	case "Secret":
		return appName + "-secret"
	case "Namespace":
		return appName + "-namespace"
	case "Ingress":
		return appName + "-ingress"
	case "StatefulSet":
		return appName + "-statefulset"
	case "PersistentVolumeClaim":
		return appName + "-pvc"
	case "DaemonSet":
		return appName + "-daemonset"
	case "CronJob":
		return appName + "-cronjob"
	case "ServiceAccount":
		return appName + "-sa"
	case "HorizontalPodAutoscaler":
		return appName + "-hpa"
	case "ResourceQuota":
		return appName + "-quota"
	case "LimitRange":
		return appName + "-limits"
	case "NetworkPolicy":
		return appName + "-netpol"
	case "Role":
		return appName + "-role"
	case "RoleBinding":
		return appName + "-rolebinding"
	case "ClusterRole":
		return appName + "-clusterrole"
	case "ClusterRoleBinding":
		return appName + "-clusterrolebinding"
	case "StorageClass":
		return appName + "-storageclass"
	default:
		return appName + "-" + strings.ToLower(kind)
	}
}

// ExtractK8sRelationships derives resource nodes and edges from the request.
func ExtractK8sRelationships(req domain.ConfigRequest) ([]domain.ResourceNode, []domain.ResourceEdge) {
	if req.Type != "k8s" && req.Type != "kustomize" {
		return nil, nil
	}
	appName := req.AppName
	resourceSet := make(map[string]bool)
	for _, r := range req.K8sResource {
		resourceSet[r] = true
	}

	var nodes []domain.ResourceNode
	for _, kind := range req.K8sResource {
		nodes = append(nodes, domain.ResourceNode{
			Kind: kind,
			Name: resourceName(kind, appName),
		})
	}

	var edges []domain.ResourceEdge

	// Build a lookup for quick kind → name resolution
	kindName := make(map[string]string)
	for _, n := range nodes {
		kindName[n.Kind] = n.Name
	}

	// Define relationship rules between resource kinds
	type edgeRule struct {
		source   string
		target   string
		relation string
		optional string // if set, only emit when this kind is also present
	}

	rules := []edgeRule{
		{source: "Deployment", target: "ConfigMap", relation: "mounts", optional: "ConfigMap"},
		{source: "Deployment", target: "Secret", relation: "mounts", optional: "Secret"},
		{source: "Deployment", target: "ServiceAccount", relation: "uses-sa", optional: "ServiceAccount"},
		{source: "Deployment", target: "Service", relation: "exposed-by", optional: "Service"},
		{source: "Ingress", target: "Service", relation: "routes-to", optional: "Service"},
		{source: "Service", target: "Deployment", relation: "selects", optional: "Deployment"},
		{source: "StatefulSet", target: "Service", relation: "governed-by", optional: "Service"},
		{source: "HorizontalPodAutoscaler", target: "Deployment", relation: "scales", optional: "Deployment"},
		{source: "RoleBinding", target: "Role", relation: "binds", optional: "Role"},
		{source: "RoleBinding", target: "ServiceAccount", relation: "grants-to", optional: "ServiceAccount"},
		{source: "ClusterRoleBinding", target: "ClusterRole", relation: "binds", optional: "ClusterRole"},
		{source: "ClusterRoleBinding", target: "ServiceAccount", relation: "grants-to", optional: "ServiceAccount"},
	}

	for _, rule := range rules {
		srcName, srcOK := kindName[rule.source]
		tgtName, tgtOK := kindName[rule.target]
		if !srcOK || !tgtOK {
			continue
		}
		if rule.optional != "" {
			if !resourceSet[rule.optional] {
				continue
			}
		}
		edges = append(edges, domain.ResourceEdge{
			SourceKind: rule.source,
			SourceName: srcName,
			TargetKind: rule.target,
			TargetName: tgtName,
			Relation:   rule.relation,
		})
	}

	return nodes, edges
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
