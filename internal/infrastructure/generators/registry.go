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
