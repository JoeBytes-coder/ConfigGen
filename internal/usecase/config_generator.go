package usecase

import (
	"time"

	"configgen/internal/domain"
)

type ConfigGenerator struct {
	generator domain.ConfigGenerator
	store     domain.Store
}

func NewConfigGenerator(generator domain.ConfigGenerator, store domain.Store) *ConfigGenerator {
	return &ConfigGenerator{generator: generator, store: store}
}

func (cg *ConfigGenerator) Prepare(req *domain.ConfigRequest) {
	cg.applyDefaults(req)
}

func (cg *ConfigGenerator) GenerateAndSave(req domain.ConfigRequest) (domain.ConfigRecord, error) {
	cg.applyDefaults(&req)

	result, err := cg.generator.Generate(req)
	if err != nil {
		return domain.ConfigRecord{}, err
	}

	rec := domain.ConfigRecord{
		Request:   req,
		Result:    result,
		CreatedAt: time.Now().UTC(),
	}

	id, err := cg.store.Save(rec)
	if err != nil {
		return domain.ConfigRecord{}, err
	}

	rec.ID = id
	return rec, nil
}

func (cg *ConfigGenerator) GetRecord(id int64) (domain.ConfigRecord, error) {
	return cg.store.Find(id)
}

func (cg *ConfigGenerator) ListRecords(offset, limit int) ([]domain.ConfigRecord, error) {
	return cg.store.List(offset, limit)
}

func (cg *ConfigGenerator) applyDefaults(req *domain.ConfigRequest) {
	if req.Type != "k8s" || len(req.K8sResource) == 0 {
		return
	}

	needsImage := false
	needsPort := false

	for _, r := range req.K8sResource {
		switch r {
		case "Deployment", "StatefulSet", "DaemonSet", "CronJob":
			needsImage = true
		case "Service", "Ingress":
			needsImage = true
			needsPort = true
		default:
			needsPort = true
		}
	}

	if needsImage {
		if req.Image == "" {
			req.Image = "nginx"
		}
		if req.Tag == "" {
			req.Tag = "latest"
		}
	}
	if needsPort && req.Port == 0 {
		req.Port = 80
	}
}
