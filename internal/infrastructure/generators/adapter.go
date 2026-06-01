package generators

import "configgen/internal/domain"

type Adapter struct{}

func NewAdapter() *Adapter {
	return &Adapter{}
}

func (a *Adapter) Generate(req domain.ConfigRequest) (domain.ConfigResult, error) {
	return GenerateConfig(req)
}
