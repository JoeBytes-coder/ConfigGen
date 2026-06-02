package generators

import "configgen/internal/domain"

type Adapter struct{}

func NewAdapter() *Adapter {
	return &Adapter{}
}

func (a *Adapter) Generate(req domain.ConfigRequest) (domain.ConfigResult, error) {
	result, err := GenerateConfig(req)
	if err != nil {
		return result, err
	}

	if req.Type == "k8s" || req.Type == "kustomize" {
		nodes, edges := ExtractK8sRelationships(req)
		result.Resources = nodes
		result.Edges = edges
	}

	return result, nil
}

func (a *Adapter) RegisteredTypes() []string {
	return RegisteredTypes()
}
