package domain

type ConfigGenerator interface {
	Generate(req ConfigRequest) (ConfigResult, error)
}
