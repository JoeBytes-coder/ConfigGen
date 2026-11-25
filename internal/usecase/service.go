package usecase

import (
	"time"

	"configgen/internal/domain"
	"configgen/internal/infrastructure/render"
)

// GeneratorService 是唯一业务入口：先解析（Resolve），再渲染（Render）。
type GeneratorService struct {
	store domain.Store
}

func NewGeneratorService(store domain.Store) *GeneratorService {
	return &GeneratorService{store: store}
}

// Plan 只解析依赖，不落库，供前端实时预览。
func (s *GeneratorService) Plan(sel domain.Selection) (domain.Plan, error) {
	return domain.Resolve(sel)
}

// Render 解析 + 渲染 + 持久化。
func (s *GeneratorService) Render(sel domain.Selection) (domain.Record, error) {
	plan, err := domain.Resolve(sel)
	if err != nil {
		return domain.Record{}, err
	}
	files, bundle, err := render.Render(plan)
	if err != nil {
		return domain.Record{}, domain.NewBadRequest("渲染失败: " + err.Error())
	}
	rec := domain.Record{
		Type:           sel.Type,
		App:            plan.App,
		Selection:      sel,
		Plan:           plan,
		Files:          files,
		Bundle:         bundle,
		CatalogVersion: domain.CatalogVersion,
		CreatedAt:      time.Now().UTC(),
	}
	id, err := s.store.Save(rec)
	if err != nil {
		return domain.Record{}, err
	}
	rec.ID = id
	return rec, nil
}

func (s *GeneratorService) GetRecord(id int64) (domain.Record, error) {
	return s.store.Find(id)
}

func (s *GeneratorService) ListRecords(offset, limit int) ([]domain.RecordSummary, int, error) {
	records, err := s.store.List(offset, limit)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.store.Count()
	if err != nil {
		return nil, 0, err
	}
	return records, total, nil
}

func (s *GeneratorService) Health() error {
	return s.store.Ping()
}
