package generators

import (
	"configgen/internal/domain"
	"fmt"
	"sync"
)

var (
	generators   = map[string]Generator{}
	generatorsMu sync.RWMutex
)

func Register(name string, gen Generator) {
	generatorsMu.Lock()
	defer generatorsMu.Unlock()
	generators[name] = gen
}

func GetGenerator(genType string) (Generator, error) {
	generatorsMu.RLock()
	gen, exists := generators[genType]
	generatorsMu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("unsupported generator type: %s", genType)
	}
	return gen, nil
}

func RegisteredTypes() []string {
	generatorsMu.RLock()
	defer generatorsMu.RUnlock()
	types := make([]string, 0, len(generators))
	for t := range generators {
		types = append(types, t)
	}
	return types
}

func GenerateConfig(req domain.ConfigRequest) (domain.ConfigResult, error) {
	gen, err := GetGenerator(req.Type)
	if err != nil {
		return domain.ConfigResult{}, err
	}

	config, err := gen.Generate(req)
	if err != nil {
		return domain.ConfigResult{}, err
	}

	return domain.ConfigResult{Type: req.Type, Config: config}, nil
}
